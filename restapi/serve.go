package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/firelizzard18/gocoding"
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"strconv"
)

var portNumber = flag.Int("p", 8083, "Set the port to listen on")

func serve_init() {
	http.HandleFunc("/", serveRESTfulHTTP)
}

func serve_main() {
	err := http.ListenAndServe(fmt.Sprintf(":%d", *portNumber), nil)
	if err != nil {
		panic(err)
	}
}

func serve_fini() {
	
}

func serveRESTfulHTTP(w http.ResponseWriter, r *http.Request) {
	var resource interface{}
	var err *notaryapi.Error
	var buf bytes.Buffer

	path, method, accept, form, err := parse(r)

	defer func() {
		switch accept {
		case "text":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		case "json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")

		case "xml":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")

		case "html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		if err != nil {
			var r *notaryapi.Error

			buf.Reset()
			r = notaryapi.Marshal(err, accept, &buf, false)
			if r != nil {
				err = r
			}
			w.WriteHeader(err.HTTPCode)
		}

		buf.WriteTo(w)
		w.Write([]byte("\n\n"))
	}()

	switch method {
	case "GET":
		resource, err = find(path)

	case "POST":
		if len(path) != 1 {
			err = notaryapi.CreateError(notaryapi.ErrorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}

		resource, err = post("/"+strings.Join(path, "/"), form)

	default:
		err = notaryapi.CreateError(notaryapi.ErrorBadMethod, fmt.Sprintf(`The HTTP %s method is not supported`, method))
		return
	}

	if err != nil {
		resource = err
	}

	alt := false
	for _, s := range form["byref"] {
		b, err := strconv.ParseBool(s)
		if err == nil {
			alt = b
			break
		}
	}

	err = notaryapi.Marshal(resource, accept, &buf, alt)
}

func parse(r *http.Request) (path []string, method string, accept string, form url.Values, err *notaryapi.Error) {
	url := strings.TrimSpace(r.URL.Path)
	path = strings.Split(url, "/")
	
	pathlen := len(path)
	lastpath := path[pathlen - 1:pathlen]
	bits := strings.Split(lastpath[0], ".")
	bitslen := len(bits)
	
	if len(bits) > 1 {
		lastpath[0], bits = strings.Join(bits[:bitslen - 1], "."), bits[bitslen - 1:bitslen]
	} else {
		bits = make([]string, 0)
	}
	
	if len(path) > 0 && len(path[0]) == 0 {
		path = path[1:]
	}
	
	if len(path) > 0 && len(path[len(path) - 1]) == 0 {
		path = path[:len(path) - 1]
	}
	
	method = r.Method
	
a:	for _, accept = range r.Header["Accept"] {
		for _, accept = range strings.Split(accept, ",") {
			accept, err = parseAccept(accept, bits)
			if err == nil {
				break a
			}
		}
	}
	
	if err != nil {
		return
	}
	
	if accept == "" {
		accept = "json"
	}
	
	e := r.ParseForm()
	if e != nil {
		err = notaryapi.CreateError(notaryapi.ErrorBadPOSTData, e.Error())
		return
	}
	
	form = r.Form
	
	return
}

func parseAccept(accept string, ext []string) (string, *notaryapi.Error) {
	switch accept {
	case "text/plain":
		if len(ext) == 1 && ext[0] != "txt" {
			return ext[0], nil
		}
		return "text", nil
		
	case "application/json", "*/*":
		return "json", nil
		
	case "application/xml", "text/xml":
		return "xml", nil
		
	case "text/html":
		return "html", nil
	}
	
	return "", notaryapi.CreateError(notaryapi.ErrorNotAcceptable, fmt.Sprintf("The specified resource cannot be returned as %s", accept))
}

var blockPtrType = reflect.TypeOf((*notaryapi.Block)(nil)).Elem()

func post(context string, form url.Values) (interface{}, *notaryapi.Error) {
	newEntry := new(notaryapi.Entry)
	format, data := form.Get("format"), form.Get("data")

	switch format {
	case "", "json":
		reader := gocoding.ReadString(data)
		err := notaryapi.UnmarshalJSON(reader, newEntry)
		if err != nil {
			return nil, notaryapi.CreateError(notaryapi.ErrorJSONUnmarshal, err.Error())
		}

	case "xml":
		err := xml.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, notaryapi.CreateError(notaryapi.ErrorXMLUnmarshal, err.Error())
		}

	default:
		return nil, notaryapi.CreateError(notaryapi.ErrorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}

	if newEntry == nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Entity to be POSTed is nil`)
	}

	newEntry.StampTime()

	blockMutex.Lock()
	err := blocks[len(blocks)-1].AddEntry(newEntry)
	blockMutex.Unlock()

	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}

	return newEntry, nil
}