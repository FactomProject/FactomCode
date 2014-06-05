package rest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"bytes"
	"text/template"
)

var htmlTmpl *template.Template

func init() {
	var err error
	
	htmlTmpl, err = template.ParseFiles("test/rest/html.gwp");
	
	if err != nil {
		panic(err)
	}
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