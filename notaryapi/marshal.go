package notaryapi

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"text/template"

	"github.com/FactomProject/dynrsrc"
	"github.com/FactomProject/gocoding"
	"github.com/FactomProject/gocoding/html"
	"github.com/FactomProject/gocoding/json"
)

var htmlTmpl *template.Template

func StartStatic(path string) (err error) {
	htmlTmpl, err = template.ParseFiles(path)
	return
}

func StartDynamic(path string, readEH func(err error)) error {
	return dynrsrc.CreateDynamicResource(path, func(data []byte) {
		var err error
		htmlTmpl, err = template.New("html").Parse(string(data))
		if err != nil {
			readEH(err)
		}
	})
}

var M = struct{ Main, Alt gocoding.Marshaller }{
	json.NewMarshaller(),
	json.NewMarshaller(),
}

var hashEncoder = M.Alt.FindEncoder(reflect.TypeOf(new(Hash)))

func init() {
	M.Alt.CacheEncoder(reflect.TypeOf(new(EBlock)), AltBlockEncoder)
	//M.Alt.CacheEncoder(reflect.TypeOf(new(Entry)), AltEntryEncoder)
}

func AltBlockEncoder(scratch [64]byte, r gocoding.Renderer, v reflect.Value) {
	v = v.Elem()

	r.StartStruct()

	r.StartElement("BlockID")
	M.Alt.MarshalValue(r, v.FieldByName("BlockID"))
	r.StopElement("BlockID")

	r.StartElement("PreviousHash")
	M.Alt.MarshalValue(r, v.FieldByName("PreviousHash"))
	r.StopElement("PreviousHash")

	r.StartElement("NumEntries")
	M.Alt.MarshalObject(r, v.FieldByName("Entries").Len())
	r.StopElement("NumEntries")

	r.StartElement("Salt")
	M.Alt.MarshalValue(r, v.FieldByName("Salt"))
	r.StopElement("Salt")

	r.StopStruct()
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
		renderer = json.RenderIndented(writer, "", "  ")

	case "json":
		renderer = json.Render(writer)

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
		renderer = html.Render(writer)

	default:
		resource = CreateError(ErrorUnsupportedMarshal, fmt.Sprintf(`"%s" is an unsupported marshalling format`, accept))
		renderer = json.Render(writer)
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
