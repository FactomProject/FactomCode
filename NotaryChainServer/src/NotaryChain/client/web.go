package main

import (
	"flag"
	"fmt"
	"strconv"
	
	"io/ioutil"
	"path/filepath"
	"text/template"
	
	"github.com/hoisie/web"
	md "github.com/russross/blackfriday"
	
	"NotaryChain/restapi"
	//"NotaryChain/pagebuilder"
	"github.com/firelizzard18/dynrsrc"
)

var mdrdr md.Renderer
var mdext int
var mainTmpl *template.Template
var portNumber *int = flag.Int("p", 8087, "Set the port to listen on")

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func buildTemplateTree() (main *template.Template, err error) {
	main, err = template.ParseFiles("app/client/page.gwp")
	if err != nil { return }
	
	_, err = main.ParseGlob("app/client/*.gwp")
	if err != nil { return }
	
	matches, err := filepath.Glob("app/client/*.md")
	if err != nil { return }
	
	err = parseMarkdownTemplates(main, matches...)
	return
}

func parseMarkdownTemplates(t *template.Template, filenames...string) error {
	for _, filename := range filenames {
		data, err := ioutil.ReadFile(filename)
		if err != nil { return err }
		
		data = md.Markdown(data, mdrdr, mdext)
		
		_, err = t.New(filepath.Base(filename)).Parse(string(data))
		if err != nil { return err }
	}
	
	return nil
}

func initTemplate(load func(string) error, filename string, markdown bool) {
	handler := func(data []byte) {
		if markdown {
			data = md.Markdown(data, mdrdr, mdext)
		}
		
		err := load(string(data))
		
		if err != nil {
			panic(err)
		}
	}
	
	err := dynrsrc.CreateDynamicResource(filename, handler)
	
	if err != nil {
		panic(err)
	}
}

func main() {
	var err error
	
	err = dynrsrc.Start(watchError, readError)
	if err != nil { panic(err) }
	
	err = dynrsrc.CreateDynamicResource("app/client", func([]byte) {
		t, err := buildTemplateTree()
		if err != nil { readError(err) }
		
		mainTmpl = t
	})
	if err != nil { panic(err) }
	
	//pagebuilder.Match(`/(home)?`, "home", handleHome, "GET", "POST")
	//pagebuilder.Match(`/entries/(\d+)`, "entry", handleEntry, "GET")
	
	//pagebuilder.Server.Config.StaticDir = "app/client/static"
	
	defer func() {
		dynrsrc.Stop()
	}()
	
	//pagebuilder.Server.Run("localhost:8087")
}

func handleHome(ctx *web.Context, matches []string) map[string]string {
	switch ctx.Params["format"] {
	case "json":
		unmarshalEntryFromJSON([]byte(ctx.Params["data"]))
		
	case "ecdsa":
	}
	
	updateEntryTimeStamp()
	
	data, err := marshalEntryToJSON()
	if err != nil { return handleError(err) }
	
	return map[string]string {
		"Title": "Home",
		"EntryJSON": string(data),
		"OnLoad": "",
	}
}

func handleEntry(ctx *web.Context, matches []string) map[string]string {
	data, err := marshalEntryToJSON()
	if err != nil { return handleError(err) }
	
	return map[string]string {
		"Title": "Edit Entry " + matches[0],
		"EntryJSON": string(data),
		"OnLoad": "",
	}
}

func handleError(err *restapi.Error) map[string]string {
	data, r := restapi.Marshal(err, "html")
	if r != nil { err = r }
	
	return map[string]string {
		"Title": "Error",
		"HTTPCode": strconv.Itoa(err.HTTPCode),
		"Content": string(data),
		"OnLoad": "",
	}
}