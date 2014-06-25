package notaryapi

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gocoding"
	"github.com/firelizzard18/gocoding/json"
	"io"
	"reflect"
	"text/template"
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

var M = struct {Main, Alt gocoding.Marshaller}{
	json.NewMarshaller(),
	json.NewMarshaller(),
}

func init() {
	M.Alt.CacheEncoder(reflect.TypeOf(new(Block)), AltBlockEncoder)
//	M.Alt.CacheEncoder(reflect.TypeOf(new(Entry)), AltEntryEncoder)
}

func AltBlockEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	value = value.Elem()
	
	renderer.StartStruct()
	
	renderer.StartElement("BlockID")
	M.Alt.MarshalValue(renderer, value.FieldByName("BlockID"))
	renderer.StopElement("BlockID")
	
	renderer.StartElement("PreviousHash")
	M.Alt.MarshalValue(renderer, value.FieldByName("PreviousHash"))
	renderer.StopElement("PreviousHash")
	
	renderer.StartElement("NumEntries")
	M.Alt.MarshalObject(renderer, value.FieldByName("Entries").Len())
	renderer.StopElement("NumEntries")
	
	renderer.StartElement("Salt")
	M.Alt.MarshalValue(renderer, value.FieldByName("Salt"))
	renderer.StopElement("Salt")
	
	renderer.StopStruct()
}

func AltEntryEncoder(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
	entry := value.Interface().(*Entry)
	
	for name, value := range entry.EncodableFields() {
		if name == "Signatures" { continue }
		renderer.StartElement(name)
		M.Alt.MarshalValue(renderer, value)
		renderer.StopElement(name)
	}
	
	renderer.StartElement("NumSignatures")
	M.Alt.MarshalObject(renderer, entry.Signatures())
	renderer.StopElement("NumSignatures")
	
	renderer.StopStruct()
}

func Marshal(resource interface{}, accept string, writer io.Writer, alt bool) (r *Error) {
	var err error
	var marshaller gocoding.Marshaller
	var renderer gocoding.Renderer
	
	if alt {
		marshaller = M.Alt
	} else {
		marshaller = M.Main
	}
	
	switch accept {
	case "text":
		renderer = json.RenderIndentedJSON(writer, "", "  ")
		
	case "json":
		renderer = json.RenderJSON(writer)
		
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
		r = Marshal(resource, "json", &buf, alt)
		if r != nil {
			return r
		}
		
		str := buf.String()
		buf.Reset()
		err := htmlTmpl.Execute(&buf, str)
		if err != nil {
			r = CreateError(ErrorJSONMarshal, err.Error())
			err = marshaller.Marshal(json.RenderJSON(writer), r)
			if err != nil {
				panic(err)
			}
		}
		
		buf.WriteTo(writer)
		return
		
	default:
		resource  = CreateError(ErrorUnsupportedMarshal, fmt.Sprintf(`"%s" is an unsupported marshalling format`, accept))
		renderer = json.RenderJSON(writer)
	}
	
	err = marshaller.Marshal(renderer, resource)
	if err != nil {
		r = CreateError(ErrorJSONMarshal, err.Error())
		err = marshaller.Marshal(renderer, r)
		if err != nil {
			panic(err)
		}
	}
	return
}