package main

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"text/template"
	
	"github.com/firelizzard18/blackfriday"
	"github.com/firelizzard18/dynrsrc"
	
	"NotaryChain/notaryapi"
)

var mdrdr blackfriday.Renderer
var mdext = 0
var mainTmpl *template.Template

func templates_init() {
	mdext |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	mdext |= blackfriday.EXTENSION_TABLES
	mdext |= blackfriday.EXTENSION_FENCED_CODE
	mdext |= blackfriday.EXTENSION_AUTOLINK
	mdext |= blackfriday.EXTENSION_STRIKETHROUGH
	mdext |= blackfriday.EXTENSION_SPACE_HEADERS
	mdext |= blackfriday.EXTENSION_PASSTHROUGH_TEMPLATE_ACTION
	
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
	mdrdr = blackfriday.HtmlRenderer(htmlFlags, "", "")
	
	err := dynrsrc.CreateDynamicResource(*appDir, func([]byte) {
		t, err := buildTemplateTree()
		if err != nil { readError(err) }
		
		mainTmpl = t
	})
	if err != nil { panic(err) }
}

func buildTemplateTree() (main *template.Template, err error) {
	funcmap := template.FuncMap{
		"tmplref": templateRef,
		"enc64": templateEncode64,
		"isValidEntryID": templateIsValidEntryId,
		"entry": templateGetEntry,
		"isValidKeyID": templateIsValidKeyId,
//		"asHash": templateAsHash,
		"key": templateGetKey,
		"entryCount": getEntryCount,
//		"isEntrySignedByKey": templateIsEntrySignedByKey,
//		"isEntrySignedByKeyHash": templateIsEntrySignedByKeyHash,
		"keysExceptEntrySigs": templateKeysExceptEntrySigs,
		"keyCount": getKeyCount,
		"mkrng": templateMakeRange,
		"atoi": func(str string) (int, error) { return strconv.Atoi(str) },
		"itoa": func(num int) string { return strconv.Itoa(num) },
		"isNil": func(val interface{}) bool { switch val.(type) { case nil: return true }; return false },
		"len": func(val interface{}) int { return reflect.ValueOf(val).Len() },
		"unnil": func(val interface{}, alt interface{}) interface{} { switch val.(type) { case nil: return alt }; return val },
	}
	
	main, err = template.New("main").Funcs(funcmap).Parse(`{{template "page.gwp" .}}`)
	if err != nil { return }
	
	_, err = main.ParseGlob(fmt.Sprint(*appDir, "/*.gwp"))
	if err != nil { return }
	
	matches, err := filepath.Glob(fmt.Sprint(*appDir, "/*.md"))
	if err != nil { return }
	
	err = parseMarkdownTemplates(main, matches...)
	return
}

func parseMarkdownTemplates(t *template.Template, filenames...string) error {
	for _, filename := range filenames {
		data, err := ioutil.ReadFile(filename)
		if err != nil { return err }
		
		data = blackfriday.Markdown(data, mdrdr, mdext)
		
		_, err = t.New(filepath.Base(filename)).Parse(string(data))
		if err != nil { return err }
	}
	
	return nil
}

func templateRef(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	
	if err := mainTmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	
	return string(buf.Bytes()), nil
}

func templateEncode64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func templateIsValidEntryId(id interface{}) bool {
	switch id.(type) {
	case int:
		num := int(reflect.ValueOf(id).Int())
		if num < 0 || num >= getEntryCount() { return false }
		return true
		
	case string:
		str := reflect.ValueOf(id).String()
		num, err := strconv.Atoi(str)
		if err != nil { return false }
		return templateIsValidEntryId(num)
	
	default:
		return false
	}
}

func templateGetEntry(idx int) (map[string]interface{}, error) {
	if idx >= getEntryCount() {
		return nil, errors.New(fmt.Sprint("Index ", idx, " out of bounds for entry array"))
	}
	
	entry := getEntry(idx)
	
	count := len(entry.Signatures())
	signatures := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		hash, err := notaryapi.CreateHash(entry.Signatures()[i].Key())
		if err != nil { return nil, err }
		
		keyCount := getKeyCount()
		var j int
		for j = 0; j < keyCount; j++ {
			key, err := notaryapi.CreateHash(getKey(j).Public())
			if err != nil { return nil, err }
			if bytes.Compare(key.Bytes, hash.Bytes) == 0 {
				break
			}
		}
		
		if j == keyCount {
			j = -1
		}
		
		signatures[i] = map[string]interface{} {
			"KeyID": j,
			"Hash": hash.Bytes,
		}
	}
	
	return map[string]interface{}{
		"ID": idx,
		"Type": entry.TypeName(),
		"Signatures": signatures,
		"TimeStamp": entry.RealTime(),
		"Data": entry.Data(),
	}, nil
}

func templateIsValidKeyId(id interface{}) bool {
	switch id.(type) {
	case int:
		num := int(reflect.ValueOf(id).Int())
		if num < 0 || num >= getKeyCount() { return false }
		return true
		
	case string:
		str := reflect.ValueOf(id).String()
		num, err := strconv.Atoi(str)
		if err != nil { return false }
		return templateIsValidKeyId(num)
	
	default:
		return false
	}
}

func templateGetKey(idx int) (map[string]interface{}, error) {
	if idx >= getKeyCount() {
		return nil, errors.New(fmt.Sprint("Index ", idx, " out of bounds for key array"))
	}
	
	key := getKey(idx)
	
	hash, err := notaryapi.CreateHash(key.Public())
	if err != nil { return nil, err }
	
	return map[string]interface{}{
		"ID": idx,
		"Type": notaryapi.KeyTypeName(key.KeyType()),
		"Hash": hash.Bytes,
	}, nil
}

func templateMakeRange(cnt int) (rng []int) {
	rng = make([]int, cnt)
	
	for i := 0; i < cnt; i++ {
		rng[i] = i
	}
	
	return rng
}

//func templateIsEntrySignedByKey(entry_id, key_id int) (bool, error) {
//	hash, err := notaryapi.CreateHash(getKey(key_id))
//	if err != nil { return false, err }
//	return templateIsEntrySignedByKeyHash(entry_id, hash)
//}
//
//func templateIsEntrySignedByKeyHash(entry_id int, hash *notaryapi.Hash) (bool, error) {
//	for _, sig := range getEntry(entry_id).Signatures() {
//		shash, err := notaryapi.CreateHash(sig.Key())
//		if err != nil { return false, err }
//		
//		if shash == hash {
//			return true, nil
//		}
//	}
//	return false, nil
//}
//
//func templateAsHash(data []byte) *notaryapi.Hash {
//	return &notaryapi.Hash{data}
//}

func templateKeysExceptEntrySigs(entry_id int) (keyIDs []int, err error) {
	sigs := getEntry(entry_id).Signatures()
	sigHashes := make([]*notaryapi.Hash, len(sigs))
	for index, sig := range sigs {
		sigHashes[index], err = notaryapi.CreateHash(sig.Key())
		if err != nil { return nil, err }
	}
	
	keyCount := getKeyCount()
	keyHashes := make([]*notaryapi.Hash, keyCount)
	for i := 0; i < keyCount; i++ {
		keyHashes[i], err = notaryapi.CreateHash(getKey(i).Public())
		if err != nil { return nil, err }
	}
	
	i := 0
	keyIDs = make([]int, keyCount)
main:
	for keyIndex, keyHash := range keyHashes {
		for _, sigHash := range sigHashes {
			if bytes.Compare(keyHash.Bytes, sigHash.Bytes) == 0 {
				continue main
			}
		}
		keyIDs[i] = keyIndex
		i++
	}
	keyIDs = keyIDs[:i]
	fmt.Println("Keys:", keyIDs)
	
	return keyIDs, nil
}







