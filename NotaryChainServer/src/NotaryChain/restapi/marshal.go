package restapi

import (
	"bytes"
	"fmt"
	
	"encoding/json"
	"encoding/xml"
	"text/template"
	
	"github.com/firelizzard18/dynrsrc"
)

var htmlTmpl *template.Template

func StartStatic() (err error) {
	htmlTmpl, err = template.ParseFiles("app/rest/html.gwp");
	return
}

func StartDynamic(readEH func(err error)) error {
	return dynrsrc.CreateDynamicResource("app/rest/html.gwp", func(data []byte) {
		var err error
		htmlTmpl, err = template.New("html").Parse(string(data))
		if err != nil { readEH(err) }
	})
}

func Marshal(resource interface{}, accept string) (data []byte, r *Error) {
	var err error
	
	switch accept {
	case "text":
		data, err = json.MarshalIndent(resource, "", "  ")
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			data, err = json.MarshalIndent(r, "", "  ")
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "json":
		data, err = json.Marshal(resource)
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			data, err = json.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "xml":
		data, err = xml.Marshal(resource)
		if err != nil {
			r = CreateError(ErrorXMLMarshal, err.Error())
			data, err = xml.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "html":
		data, r = Marshal(resource, "json")
		if r != nil {
			return nil, r
		}
		
		var buf bytes.Buffer
		err := htmlTmpl.Execute(&buf, string(data))
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			data, err = json.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		
		data = buf.Bytes()
		return
	}
	
	r  = CreateError(ErrorUnsupportedMarshal, fmt.Sprintf(`"%s" is an unsupported marshalling format`, accept))
	data, err = json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return
}