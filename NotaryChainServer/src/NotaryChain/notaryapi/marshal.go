package notaryapi

import (
	"bytes"
	"fmt"
	"io"
	
	"encoding/xml"
	"text/template"
	
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gocoding"
	"github.com/firelizzard18/gocoding/json"
)

var htmlTmpl *template.Template

func StartStatic(path string) (err error) {
	htmlTmpl, err = template.ParseFiles(path);
	return
}

func StartDynamic(path string, readEH func(err error)) error {
	return dynrsrc.CreateDynamicResource(path, func(data []byte) {
		var err error
		htmlTmpl, err = template.New("html").Parse(string(data))
		if err != nil { readEH(err) }
	})
}

var marshaller = gocoding.NewMarshaller(json.JSONEncoding, nil)

func Marshal(resource interface{}, accept string, writer io.Writer) (r *Error) {
	var err error
	
	switch accept {
	case "text":
		marshaller.SetRenderer(json.RenderIndentedJSON(writer, "", "  "))
		err = marshaller.Marshal(resource)
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			err = marshaller.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "json":
		marshaller.SetRenderer(json.RenderJSON(writer))
		err = marshaller.Marshal(resource)
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			err = marshaller.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "xml":
		data, err := xml.Marshal(resource)
		if err != nil {
			r = CreateError(ErrorXMLMarshal, err.Error())
			data, err = xml.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		writer.Write(data)
		return
		
	case "html":
		var buf bytes.Buffer
		r = Marshal(resource, "json", &buf)
		if r != nil {
			return r
		}
		
		str := buf.String()
		buf.Reset()
		err := htmlTmpl.Execute(&buf, str)
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			marshaller.SetRenderer(json.RenderJSON(writer))
			err = marshaller.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		
		buf.WriteTo(writer)
		return
	}
	
	r  = CreateError(ErrorUnsupportedMarshal, fmt.Sprintf(`"%s" is an unsupported marshalling format`, accept))
	marshaller.SetRenderer(json.RenderJSON(writer))
	err = marshaller.Marshal(r)
	if err != nil {
		panic(err)
	}
	return
}