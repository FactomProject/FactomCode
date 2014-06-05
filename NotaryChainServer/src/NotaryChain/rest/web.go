package main 

import (
	"flag"
	"net/http"
	"net/url"
	ncdata "NotaryChain/data"
	"encoding/json"
	"encoding/xml"
	"text/template"
	"strconv"
	"fmt"
	"strings"
	"reflect"
	"time"
	"errors"
	"sync"
	"bytes"
	"io/ioutil"
)

var portNumber *int = flag.Int("p", 8083, "Set the port to listen on")

var blocks []*ncdata.Block
var htmlTmpl *template.Template

func load() {
	source, err := ioutil.ReadFile("/Users/firelizzard/Documents/Programming/NotaryChain/NotaryChainServer/src/NotaryChain/rest/store.json")
	if err != nil { panic(err) }
	
	if err := json.Unmarshal(source, &blocks); err != nil { panic(err) }
	
	for i := 0; i < len(blocks); i = i + 1 {
		if uint64(i) != blocks[i].BlockID {
			panic(errors.New("BlockID does not equal index"))
		}
	}
	ncdata.UpdateNextBlockID(uint64(len(blocks)))
	
	htmlTmpl, err = template.ParseFiles("/Users/firelizzard/Documents/Programming/NotaryChain/NotaryChainServer/src/NotaryChain/rest/html.gwp");
	
	if err != nil {
		panic(err)
	}
	
	ticker := time.NewTicker(time.Minute * 5)
	defer func() {
		ticker.Stop()
	}()
	go func() {
		for _ = range ticker.C {
			notarize()
		}
	}()
}

var blockMutex = &sync.Mutex{}

func main() {
	load()
	
	http.HandleFunc("/", serveRESTfulHTTP)
	http.ListenAndServe(":" + strconv.Itoa(*portNumber), nil)
}

func notarize() {
	fmt.Println("Checking if should send current block")
	blockMutex.Lock()
	fmt.Println("Sending block, creating new block")
	blockMutex.Unlock()
}

func serveRESTfulHTTP(w http.ResponseWriter, r *http.Request) {
	var resource interface{}
	var data []byte
	var err *restError
	
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
			var r *restError
			
			data, r = marshal(err, accept)
			if r != nil {
				err = r
			}
			w.WriteHeader(err.HTTPCode)
		}
		
		w.Write(data)
		w.Write([]byte("\n\n"))
	}()
	
	switch method {
	case "GET":
		resource, err = find(path)
		
	case "POST":
		if len(path) != 1 {
			err = createError(errorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}
		
		resource, err = post("/" + strings.Join(path, "/"), form)
		
	default:
		err = createError(errorBadMethod, fmt.Sprintf(`The HTTP %s method is not supported`, method))
		return
	}
	
	data, err = marshal(resource, accept)
}

var blockPtrType = reflect.TypeOf((*ncdata.Block)(nil)).Elem()

func post(context string, form url.Values) (interface{}, *restError) {
	newEntry := new(ncdata.PlainEntry)
	format, data := form.Get("format"), form.Get("data")
	
	switch format {
	case "", "json":
		err := json.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, createError(errorJSONUnmarshal, err.Error())
		}
		
	case "xml":
		err := xml.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, createError(errorXMLUnmarshal, err.Error())
		}
	
	default:
		return nil, createError(errorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}
	
	if newEntry == nil {
		return nil, createError(errorInternal, `Entity to be POSTed is nil`)
	}
	
	newEntry.TimeStamp = time.Now().Unix()
	
	blockMutex.Lock()
	err := blocks[len(blocks) - 1].AddEntry(newEntry)
	blockMutex.Unlock()
	
	if err != nil {
		return nil, createError(errorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}
	
	return newEntry, nil
}

func marshal(resource interface{}, accept string) (data []byte, r *restError) {
	var err error
	
	switch accept {
	case "text":
		data, err = json.MarshalIndent(resource, "", "  ")
		if err != nil {
			r = createError(errorJSONMarshal, err.Error())
			data, err = json.MarshalIndent(r, "", "  ")
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "json":
		data, err = json.Marshal(resource)
		if err != nil {
			r = createError(errorJSONMarshal, err.Error())
			data, err = json.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "xml":
		data, err = xml.Marshal(resource)
		if err != nil {
			r = createError(errorXMLMarshal, err.Error())
			data, err = xml.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		return
		
	case "html":
		data, r = marshal(resource, "json")
		if r != nil {
			return nil, r
		}
		
		var buf bytes.Buffer
		err := htmlTmpl.Execute(&buf, string(data))
		if err != nil {
			r = createError(errorJSONMarshal, err.Error())
			data, err = json.Marshal(r)
			if err != nil {
				panic(err)
			}
		}
		
		data = buf.Bytes()
		return
	}
	
	r  = createError(errorUnsupportedMarshal, fmt.Sprintf(`"%s" is an unsupported marshalling format`, accept))
	data, err = json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return
}
