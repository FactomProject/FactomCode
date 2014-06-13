package main

import (
	"bytes"
	"fmt"
	
	"github.com/hoisie/web"
	
	"NotaryChain/restapi"
)

var server = web.NewServer()

func serve_init() {
	server.Config.StaticDir = fmt.Sprint(*appDir, "/static")
	
	server.Get(`/(?:home)?`, handleHome)
	server.Get(`/entries/?`, handleEntries)
	server.Post(`/entries/?`, handleEntriesPost)
	server.Get(`/entries/(\d+)`, handleEntry)
	server.Get(`/keys/?`, handleKeys)
	server.Post(`/keys/?`, handleKeysPost)
	server.Get(`/keys/(\d+)`, handleKey)
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
		"EntryID": "",
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleEntriesPost(ctx *web.Context) {
	handleEntries(ctx)
}

func handleEntry(ctx *web.Context, id string) {
	err := safeWrite(ctx, 200, map[string]interface{} {
		"Title": fmt.Sprint("Entry ", id),
		"ContentTmpl": "entry.gwp",
		"ContentWith": "",
		"EntryID": id,
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleKeys(ctx *web.Context) {
	err := safeWrite(ctx, 200, map[string]interface{} {
		"Title": "Keys",
		"ContentTmpl": "keys.gwp",
		"KeyID": "",
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleKeysPost(ctx *web.Context) {
	handleKeys(ctx)
}

func handleKey(ctx *web.Context, id string) {
	err := safeWrite(ctx, 200, map[string]interface{} {
		"Title": fmt.Sprint("Key ", id),
		"ContentTmpl": "key.gwp",
		"ContentWith": "",
		"KeyID": id,
	})
	if err != nil {
		handleError(ctx, err)
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