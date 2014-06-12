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
	server.Get(`/entities/?`, handleEntities)
	server.Post(`/entities/?`, handleEntitiesPost)
	server.Get(`/entities/(\d+)`, handleEntity)
	server.Get(`/keys/?`, handleKeys)
	server.Post(`/keys/?`, handleKeysPost)
	server.Get(`/keys/(\d+)`, handleKey)
}

func safeWrite(ctx *web.Context, code int, data map[string]interface{}) *restapi.Error {
	var buf bytes.Buffer
	
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
		"IsHome": true,
	})
	if err != nil {
		handleError(ctx, err)
	}
}

func handleEntities(ctx *web.Context) {
	
}

func handleEntitiesPost(ctx *web.Context) {
	
}

func handleEntity(ctx *web.Context, id string) {
	
}

func handleKeys(ctx *web.Context) {
	
}

func handleKeysPost(ctx *web.Context) {
	
}

func handleKey(ctx *web.Context, id string) {
	
}


func handleError(ctx *web.Context, err *restapi.Error) {
	data, r := restapi.Marshal(err, "html")
	if r != nil { err = r }
	
	err = safeWrite(ctx, err.HTTPCode, map[string]interface{} {
		"Title": "Error",
		"HTTPCode": err.HTTPCode,
		"Content": string(data),
		"ContentTmpl": "error.gwp",
	})
	if err != nil {
		handleFail(ctx)
	}
}

func handleFail(ctx *web.Context) {
	ctx.WriteHeader(500)
	ctx.Write([]byte("<!DOCTYPE html><html><head><title>Server Failure</title></head><body>Something is seriously broken</body></html>"))
}