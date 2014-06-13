package main

import (
	"bytes"
	"fmt"
	"strconv"
	
	"github.com/hoisie/web"
	
	"NotaryChain/restapi"
)

var server = web.NewServer()

func serve_init() {
	server.Config.StaticDir = fmt.Sprint(*appDir, "/static")
	
	server.Get(`/(?:home)?`, handleHome)
	server.Get(`/entries/?`, handleEntries)
	server.Post(`/entries/?`, handleEntriesPost)
	server.Get(`/entries/(\d+)(?:/(\w+))?`, handleEntry)
	server.Get(`/keys/?`, handleKeys)
	
	server.Post(`/keys/?`, handleKeysPost)
	server.Get(`/keys/(\d+)(?:/(\w+))?`, handleKey)
}

func safeWrite(ctx *web.Context, code int, data map[string]interface{}) *restapi.Error {
	var buf bytes.Buffer
	
	data["EntryCount"] = getEntryCount()
	data["KeyCount"] = getEntryCount()
	
	err := mainTmpl.Execute(&buf, data)
	if err != nil { return restapi.CreateError(restapi.ErrorTemplateError, err.Error()) }
	
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

func handleEntriesPost(ctx *web.Context) {
	handleEntries(ctx)
}

func handleEntry(ctx *web.Context, id string, action string) {
	var title string
	
	idx, err := strconv.Atoi(id)
	
	if err == nil {
		title = fmt.Sprint("Entry ", idx)
	} else {
		title = "Entry not found"
		idx = -1
	}
	
	r := safeWrite(ctx, 200, map[string]interface{} {
		"Title": title,
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

func handleKeysPost(ctx *web.Context) {
	handleKeys(ctx)
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


func handleError(ctx *web.Context, err *restapi.Error) {
	data, r := restapi.Marshal(err, "json")
	if r != nil { err = r }
	
	err = safeWrite(ctx, err.HTTPCode, map[string]interface{} {
		"Title": "Error",
		"HTTPCode": err.HTTPCode,
		"Content": string(data),
		"ContentTmpl": "error.gwp",
	})
	if err != nil {
		handleFail(ctx, err)
	}
}

func handleFail(ctx *web.Context, err error) {
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