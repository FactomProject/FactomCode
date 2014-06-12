package main

import (
	"bytes"
	"fmt"
	
	"io/ioutil"
	"path/filepath"
	"text/template"
	
	"github.com/firelizzard18/blackfriday"
	"github.com/firelizzard18/dynrsrc"
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

