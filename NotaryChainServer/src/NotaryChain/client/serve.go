package main

import (
	"bytes"
	"fmt"
	"strconv"
	
	"encoding/base64"
	
	"github.com/hoisie/web"
	
	"NotaryChain/notaryapi"
)

var server = web.NewServer()

func serve_init() {
	server.Config.StaticDir = fmt.Sprint(*appDir, "/static")
	
	server.Get(`/failed`, handleFailed)
	server.Get(`/(?:home)?`, handleHome)
	server.Get(`/entries/?`, handleEntries)
	server.Get(`/entries/add`, handleAddEntry)
	server.Get(`/entries/(\d+)(?:/(\w+)(?:/(\d+))?)?`, handleEntry)
	server.Get(`/keys/?`, handleKeys)
	server.Get(`/keys/(\d+)(?:/(\w+))?`, handleKey)
	
	server.Post(`/entries/?`, handleEntriesPost)
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

func handleHome(ctx *web.Context) {
	err := safeWrite(ctx, 200, map[string]interface{} {
		"Title": "Home",
		"ContentTmpl": "home.md",
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleEntries(ctx *web.Context) {
	err := safeWrite(ctx, 200, map[string]interface{} {
		"Title": "Entries",
		"ContentTmpl": "entries.gwp",
		"AddEntry": true,
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleAddEntry(ctx *web.Context) {
	
}

func handleEntry(ctx *web.Context, id string, action string, aid string) {
	idx, err := strconv.Atoi(id)
	if err != nil  { idx = -1 }
	
	aidx, _ := strconv.Atoi(aid)
	
	if action == "rmsig" && templateIsValidEntryId(id) {
		entry := getEntry(idx)
		if entry.Unsign(aidx) {
			ctx.Header().Add("Location", fmt.Sprint("/entries/", idx))
			ctx.WriteHeader(303)
			return
		}
	}
	
	r := safeWrite(ctx, 200, map[string]interface{} {
		"Title": fmt.Sprint("Entry ", idx),
		"ContentTmpl": "entry.gwp",
		"EntryID": idx,
		"Edit": action == "edit",
		"ShowEntries": true,
	})
	if r != nil {
		handleError(ctx, r)
	}
}

func handleKeys(ctx *web.Context) {
	err := safeWrite(ctx, 200, map[string]interface{} {
		"Title": "Keys",
		"ContentTmpl": "keys.gwp",
		"AddKey": true,
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleKey(ctx *web.Context, id string, action string) {
	var title string
	
	idx, err := strconv.Atoi(id)
	
	if err == nil {
		title = fmt.Sprint("Key ", idx)
	} else {
		title = "Key not found"
		idx = -1
	}
	
	r := safeWrite(ctx, 200, map[string]interface{} {
		"Title": title,
		"ContentTmpl": "key.gwp",
		"EntryID": idx,
		"Edit": action == "edit",
		"ShowKeys": true,
	})
	if r != nil {
		handleError(ctx, r)
	}
}

func handleEntriesPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()
	
	switch ctx.Params["action"] {
	case "editDataEntry":
		id := ctx.Params["id"]
		idx, err := strconv.Atoi(id)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing id: ", err.Error())
			abortReturn = fmt.Sprint("/entries")
			return
		}
		if !templateIsValidEntryId(idx) {
			abortMessage = fmt.Sprint("Failed to edit data entry data: bad id: ", id)
			abortReturn = fmt.Sprint("/entries")
			return
		}
		
		entry := getEntry(idx)
		data, err := base64.StdEncoding.DecodeString(ctx.Params["data"])
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing data: ", err.Error())
			abortReturn = fmt.Sprint("/entries/", id)
			return
		}
		
		entry.UpdateData(data)
	}
}

func handleError(ctx *web.Context, err *notaryapi.Error) {
	data, r := notaryapi.Marshal(err, "json")
	if r != nil { err = r }
	
	err = safeWrite(ctx, err.HTTPCode, map[string]interface{} {
		"Title": "Error",
		"HTTPCode": err.HTTPCode,
		"Content": string(data),
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

func handleFailed(ctx *web.Context) {
}