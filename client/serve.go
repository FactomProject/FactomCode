package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/firelizzard18/gobundle"
	"github.com/firelizzard18/gocoding"
	"github.com/hoisie/web"
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"net/http"
	"net/url"
	"strconv"
)

var server = web.NewServer()

func serve_init() {
	server.Config.StaticDir = gobundle.DataFile("/static")
	
	server.Get(`/(?:home)?`, handleHome)
	server.Get(`/entries/?`, handleEntries)
	server.Get(`/keys/?`, handleKeys)
	server.Get(`/explore((?:/(?:\w|\d)+)+)?`, handleExplore)
	server.Get(`/settings/?`, handleSettings)
	server.Get(`/failed`, handleFailed)
	
	server.Post(`/entries/?`, handleEntriesPost)
	server.Post(`/keys/?`, handleKeysPost)
	server.Post(`/settings/?`, handleSettingsPost)
	
	server.Get(`/entries/(?:add|\+)`, handleAddEntry)
	server.Get(`/entries/([^/]+)(?:/([^/]+)(?:/([^/]+))?)?`, handleEntry)
	server.Get(`/keys/(?:add|\+)`, handleAddKey)
	server.Get(`/keys/([^/]+)(?:/([^/]+))?`, handleKey)
}

func safeWrite(ctx *web.Context, code int, data map[string]interface{}) *notaryapi.Error {
	var buf bytes.Buffer
	
	data["EntryCount"] = getEntryCount()
	data["KeyCount"] = getEntryCount()
	
	err := mainTmpl.Execute(&buf, data)
	if err != nil { return notaryapi.CreateError(notaryapi.ErrorTemplateError, err.Error()) }
	
	ctx.WriteHeader(code)
	ctx.Write(buf.Bytes())
	
	return nil
}

func safeWrite200(ctx *web.Context, data map[string]interface{}) {
	err := safeWrite(ctx, 200, data)
	if err != nil {
		handleError(ctx, err)
	}
}

func handleHome(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Home",
		"ContentTmpl": "home.md",
	})
}

func handleEntries(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Entries",
		"ContentTmpl": "entries.gwp",
		"AddEntry": true,
	})
}

func handleExplore(ctx *web.Context, rest string) {
	if rest == "" || rest == "/" {
		safeWrite200(ctx, map[string]interface{} {
			"Title": "Explore",
			"ContentTmpl": "exploreintro.md",
		})
		return
	}
	
	url := fmt.Sprintf(`http://%s/v1/blocks%s?byref=true`, Settings.Server, rest)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorHTTPNewRequestFailure, err.Error())); return }
	
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorHTTPDoRequestFailure, err.Error())); return }
	
	data := &map[string]interface{} {}
	err = safeUnmarshal(gocoding.Read(resp.Body, 1024), data)
	resp.Body.Close()
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorJSONUnmarshal, err.Error())); return }
	
	if num, ok := (*data)["NumEntries"]; ok {
		links := make([]*link, num.(int64))
		for i := int64(0); i < num.(int64); i++ {
			links[i] = &link{fmt.Sprintf(`Entry %d`, i), fmt.Sprintf(`/explore%s/entries/%d`, rest, i)}
		}
		(*data)["Entries"] = links
		delete(*data, "NumEntries")
	}
	
	buf := new(bytes.Buffer); //buf.ReadFrom(resp.Body)
	err = safeMarshalHTML(buf, data)
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorHTMLMarshal, err.Error())); return }
	
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Explore",
		"ContentTmpl": "explore.md",
		"Data": buf.String(),
	})
}

func handleSettings(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Settings",
		"ContentTmpl": "settings.gwp",
		"Server": Settings.Server,
	})
}

func handleKeys(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Keys",
		"ContentTmpl": "keys.gwp",
		"AddKey": true,
	})
}

func handleFailed(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Failure",
		"ContentTmpl": "failed.md",
		"Message": ctx.Params["message"],
		"Return": ctx.Params["return"],
	})
}

func handleEntriesPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		} else if abortReturn != "" {
			ctx.Header().Add("Location", abortReturn)
			ctx.WriteHeader(303)
		}
	}()
	
	switch action := ctx.Params["action"]; action {
	case "editDataEntry":
		id := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", id)
		idx, err := strconv.Atoi(id)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(idx) {
			abortMessage = fmt.Sprint("Failed to edit data entry data: bad id: ", id)
			return
		}
		
		entry := getEntry(idx)
		data, err := hex.DecodeString(ctx.Params["data"])
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing data: ", err.Error())
			return
		}
		
		entry.EntryData = notaryapi.NewPlainData(data)
		
		storeEntry(idx)
		
	case "submitEntry":
		id := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", id)
		idx, err := strconv.Atoi(id)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to submit entry: error parsing id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(idx) {
			abortMessage = fmt.Sprint("Failed to submit entry: bad id: ", id)
			return
		}
//		if sub := getEntrySubmission(idx); sub != nil {
//			abortMessage = fmt.Sprint("Failed to submit entry: entry has already been submitted to ", sub.Host)
//			return
//		}
		
		entry := getEntry(idx)
		buf := new(bytes.Buffer)
		err = safeMarshal(buf, entry)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to submit entry: entry could not be marshalled: ", err.Error())
			return
		}
		
		server := fmt.Sprintf(`http://%s/v1`, Settings.Server)
		data := url.Values{}
		data.Set("format", "json")
		data.Set("data", buf.String())
		
		resp, err := http.PostForm(server, data)
		if err != nil {
			abortMessage = fmt.Sprint("An error occured while submitting the entry (entry may have been accepted by the server but was not locally flagged as such): ", err.Error())
			return
		}
		resp.Body.Close()
		
		flagSubmitEntry(idx)
		
	
	case "rmEntrySig":
		entry_id_str := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", entry_id_str)
		entry_id, err := strconv.Atoi(entry_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to remove entry signature: error parsing entry id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(entry_id) {
			abortMessage = fmt.Sprint("Failed to remove entry signature: bad entry id: ", entry_id_str)
			return
		}
		entry := getEntry(entry_id)
		
		sig_id_str := ctx.Params["sig_id"]
		sig_id, err := strconv.Atoi(sig_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to remove entry signature: error parsing signature id: ", err.Error())
			return
		}
		if sig_id >= len(entry.Signatures()){
			abortMessage = fmt.Sprint("Failed to remove entry signature: bad entry signature id: ", sig_id_str)
			break
		}
		
		if entry.Unsign(sig_id) {
			ctx.Header().Add("Location", abortReturn)
			ctx.WriteHeader(303)
		} else {
			abortMessage = fmt.Sprint("Failed to remove entry signature #", sig_id)
		}
		
		storeEntry(entry_id)
		
	case "signEntry":
		entry_id_str := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", entry_id_str)
		entry_id, err := strconv.Atoi(entry_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to sign entry: error parsing entry id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(entry_id) {
			abortMessage = fmt.Sprint("Failed to sign entry: bad entry id: ", entry_id_str)
			return
		}
		entry := getEntry(entry_id)
		
		key_id_str := ctx.Params["key"]
		key_id, err := strconv.Atoi(key_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to sign entry: error parsing key id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(entry_id) {
			abortMessage = fmt.Sprint("Failed to sign entry: bad key id: ", key_id_str)
			return
		}
		var key notaryapi.PrivateKey
		var ok bool
		_key := getKey(key_id)
		if key, ok = _key.(notaryapi.PrivateKey); !ok {
			abortMessage = fmt.Sprint("Failed to sign entry: key with id ", key_id_str, " is not a private key")
			return
		}
		
		err = entry.Sign(rand.Reader, key)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to sign entry: error while signing: ", err.Error())
			return
		}
		
		storeEntry(entry_id)
		
	case "genEntry":
		abortReturn = fmt.Sprint("/entries/add")
		
		sid := ctx.Params["datatype"]
		id, err := strconv.Atoi(sid)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to generate entry: error data type: ", err.Error())
			return
		}
		if id != notaryapi.PlainDataType {
			abortMessage = fmt.Sprint("Failed to generate entry: unsupported data type: ", id)
			return
		}
		
		entry := notaryapi.NewDataEntry([]byte{})
		addEntry(entry)
		
		ctx.Header().Add("Location", fmt.Sprintf("/entries/%d", Settings.NextEntryID-1))
		ctx.WriteHeader(303)
		
	default:
		abortReturn = fmt.Sprint("/entries")
		abortMessage = fmt.Sprint("Unknown action: ", action)
	}
}

func handleKeysPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()
	
	switch ctx.Params["action"] {
	case "genKey":
		abortReturn = fmt.Sprint("/keys/add")
		
		sid := ctx.Params["algorithm"]
		id, err := strconv.Atoi(sid)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to generate key: error parsing algorithm id: ", err.Error())
			return
		}
		
		key, err := notaryapi.GenerateKeyPair(int8(id), rand.Reader)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to generate key: error generating key: ", err.Error())
			return
		}
		if key == nil {
			abortMessage = fmt.Sprint("Failed to generate key: unsupported algorithm id: ", id)
			return
		}
		
		addKey(key)
		
		ctx.Header().Add("Location", fmt.Sprintf("/keys/%d", Settings.NextKeyID-1))
		ctx.WriteHeader(303)
	}
}

func handleSettingsPost(ctx *web.Context) {
	Settings.Server = ctx.Params["server"]
	saveSettings()
	handleSettings(ctx)
}

func handleAddEntry(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Add Entry",
		"ContentTmpl": "addentry.gwp",
	})
}

func handleEntry(ctx *web.Context, entry_id_str string, action string, action_id_str string) {
	var err error
	var title, error_str, tmpl string
	var entry_id int
	
	defer func(){
		if action == "submit" {
			tmpl = "entrysubmit.gwp"
		} else {
			tmpl = "entry.gwp"
		}
		
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"ContentTmpl": tmpl,
			"EntryID": entry_id,
			"Error": error_str,
			"Mode": action,
			"ShowEntries": true,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	entry_id, err = strconv.Atoi(entry_id_str)
	
	if err != nil  {
		error_str = fmt.Sprintf("Bad entry id: %s", err.Error())
		entry_id = -1
		title = "Entry not found"
		return
	} else {
		title = fmt.Sprint("Entry ", entry_id)
	}
	
	switch action {
	case "", "edit", "sign", "submit":
		return
		
	default:
		error_str = fmt.Sprintf("Unknown action: %s", action)
	}
}

func handleAddKey(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Add Key",
		"ContentTmpl": "addkey.gwp",
	})
}

func handleKey(ctx *web.Context, key_id_str string, action string) {
	var err error
	var title, error_str string
	var key_id int
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"ContentTmpl": "key.gwp",
			"KeyID": key_id,
			"Edit": action == "edit",
			"Error": error_str,
			"ShowKeys": true,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	key_id, err = strconv.Atoi(key_id_str)
	
	if err != nil  {
		error_str = fmt.Sprintf("Bad key id: %s", err.Error())
		key_id = -1
		title = "Key not found"
		return
	} else {
		title = fmt.Sprint("Key ", key_id)
	}
	
	switch {
	case action == "" || action == "edit":
		return
	
	default:
		error_str = fmt.Sprintf("Unknown action: %s", action)
	}
}

func handleError(ctx *web.Context, err *notaryapi.Error) {
	var buf bytes.Buffer
	
	r := notaryapi.Marshal(err, "json", &buf, false)
	if r != nil { err = r }
	
	err = safeWrite(ctx, err.HTTPCode, map[string]interface{} {
		"Title": "Error",
		"HTTPCode": err.HTTPCode,
		"Content": buf.String(),
		"ContentTmpl": "error.gwp",
	})
	if err != nil {
		str := fmt.Sprintf(`<!DOCTYPE html>
	<html>
		<head>
			<title>Server Failure</title>
		</head>
		<body>
			Something is seriously broken.
			<pre>%s</pre>
		</body>
	</html>`, err.Error())
		
		ctx.WriteHeader(500)
		ctx.Write([]byte(str))
	}
}