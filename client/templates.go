package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/firelizzard18/blackfriday"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gobundle"
	"github.com/FactomProject/FactomCode/notaryapi"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strconv"
	"text/template"
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
	
	err := dynrsrc.CreateDynamicResource(*gobundle.Setup.Directories.Data, func([]byte) {
		t, err := buildTemplateTree()
		if err != nil { readError(err) }
		
		mainTmpl = t
	})
	if err != nil { panic(err) }
}

func buildTemplateTree() (main *template.Template, err error) {
	funcmap := template.FuncMap{
		"tmplref": templateRef,
		"enc64": base64.StdEncoding.EncodeToString,
		"atoi": func(str string) (int, error) { return strconv.Atoi(str) },
		"itoa": func(num int) string { return strconv.Itoa(num) },
		"isNil": func(val interface{}) bool { switch val.(type) { case nil: return true }; return false },
		"len": func(val interface{}) int { return reflect.ValueOf(val).Len() },
		"unnil": func(val interface{}, alt interface{}) interface{} { switch val.(type) { case nil: return alt }; return val },
		
		"entry": templateGetEntry,
		"activeEntryIDs": getActiveEntryIDs,
		"pendingEntryIDs": getPendingEntryIDs,
		"confirmedEntryIDs": getConfirmedEntryIDs,
		"isValidEntryID": templateIsValidEntryId,
		"isEntrySubmitted": func (idx int) bool { sub := getEntrySubmission(idx); return sub != nil && sub.Host != "" },
		"entrySubmission": getEntrySubmission,
		
		"key": templateGetKey,
		"keyIDs": getKeyIDs,
		"keyCount": getKeyCount,
		"isValidKeyID": templateIsValidKeyId,
		
		"keysExceptEntrySigs": templateKeysExceptEntrySigs,
		//"eBlock": getEBlock,
	}
	
	main, err = template.New("main").Funcs(funcmap).Parse(`{{template "page.gwp" .}}`)
	if err != nil { return }
	
	_, err = main.ParseGlob(gobundle.DataFile("/*.gwp"))
	if err != nil { return }
	
	matches, err := filepath.Glob(gobundle.DataFile("/*.md"))
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

func templateGetEntry(idx int) (map[string]interface{}, error) {

	fentry, ok := entries[idx]
	if !ok {
		return nil, errors.New(fmt.Sprint("No entry at index", idx))
	}
	entry := fentry.Entry
	
	count := len(entry.Signatures())
	signatures := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		hash, err := notaryapi.CreateHash(entry.Signatures()[i].Key())
		if err != nil { return nil, err }
		
		j := -1
		for id, key := range keys{
			key, err := notaryapi.CreateHash(key.Public())
			if err != nil { return nil, err }
			if bytes.Compare(key.Bytes, hash.Bytes) == 0 {
				j = id
				break
			}
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
		"Data": entry.EncodeToString(),
		"Submitted": fentry.Submitted,
	}, nil
}

func templateIsValidEntryId(id interface{}) bool {
	switch id.(type) {
	case int:
		num := int(reflect.ValueOf(id).Int())
		_, ok := entries[num]
		return ok
		
	case string:
		str := reflect.ValueOf(id).String()
		num, err := strconv.Atoi(str)
		if err != nil { return false }
		return templateIsValidEntryId(num)
	
	default:
		return false
	}
}

func templateGetKey(idx int) (map[string]interface{}, error) {
	key, ok := keys[idx]
	if !ok {
		return nil, errors.New(fmt.Sprint("No key at index", idx))
	}
	
	hash, err := notaryapi.CreateHash(key.Public())
	if err != nil { return nil, err }
	
	return map[string]interface{}{
		"ID": idx,
		"Type": notaryapi.KeyTypeName(key.KeyType()),
		"Hash": hash.Bytes,
	}, nil
}

func templateIsValidKeyId(id interface{}) bool {
	switch id.(type) {
	case int:
		num := int(reflect.ValueOf(id).Int())
		_, ok := keys[num]
		return ok
		
	case string:
		str := reflect.ValueOf(id).String()
		num, err := strconv.Atoi(str)
		if err != nil { return false }
		return templateIsValidKeyId(num)
	
	default:
		return false
	}
}

func templateKeysExceptEntrySigs(entry_id int) (keyIDs []int, err error) {
	sigs := getEntry(entry_id).Signatures()
	sigHashes := make([]*notaryapi.Hash, len(sigs))
	for index, sig := range sigs {
		sigHashes[index], err = notaryapi.CreateHash(sig.Key())
		if err != nil { return nil, err }
	}
	
	keyHashes := make(map[int]*notaryapi.Hash)
	for id, key := range keys {
		keyHashes[id], err = notaryapi.CreateHash(key.Public())
		if err != nil { return nil, err }
	}
	
	i := 0
	keyIDs = make([]int, len(keyHashes))
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
	
	return keyIDs, nil
}
