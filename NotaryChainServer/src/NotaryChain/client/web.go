package main

import (
	"flag"
	"net/http"
	"text/template"
	"strconv"
	"sync"
	"time"
	"encoding/json"
	ncdata "NotaryChain/data"
	ncrest "NotaryChain/rest"
)

var portNumber *int = flag.Int("p", 8087, "Set the port to listen on")
var homeTmpl, editJSONTmpl *template.Template
var entry *ncdata.PlainEntry

func init() {
	var err error
	
	entry = new(ncdata.PlainEntry)
	entry.StructuredData = make([]byte, 0)
	entry.Signatures = make([]*ncdata.Signature, 0)
	
	homeTmpl, err = template.ParseFiles("test/client/home.gwp");
	if err != nil { panic(err) }
	
	editJSONTmpl, err = template.ParseFiles("test/client/json.gwp");
	if err != nil { panic(err) }
}

var blockMutex = &sync.Mutex{}

func main() {
	http.HandleFunc("/", serveTemplate(runHome))
	http.HandleFunc("/json", serveTemplate(runEditJSON))
	http.ListenAndServe(":" + strconv.Itoa(*portNumber), nil)
}

func serveTemplate(run func(w http.ResponseWriter, r *http.Request) *ncrest.Error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := run(w, r); err != nil {
			data, r := ncrest.Marshal(err, "html")
			if r != nil {
				err = r
			}
			w.WriteHeader(err.HTTPCode)
			w.Write(data)
		}
		
		w.Write([]byte("\n\n"))
	}
}

func runHome(w http.ResponseWriter, r *http.Request) *ncrest.Error {
	if r.URL.Path != "/" {
		w.WriteHeader(404)
		return nil
	}
	
	data := []byte(r.Form.Get("data"))
	
	if len(data) > 0 {
		err := json.Unmarshal(data, entry)
		if err != nil {
			return ncrest.CreateError(ncrest.ErrorJSONUnmarshal, err.Error())
		}
	}
	
	entry.TimeStamp = time.Now().Unix()
	
	data, err := ncrest.Marshal(entry, "json")
	if err != nil { return nil }
	
	if err := homeTmpl.Execute(w, string(data)); err != nil {
		w.WriteHeader(500)
	}
	
	return nil
}

func runEditJSON(w http.ResponseWriter, r *http.Request) *ncrest.Error {
	if r.URL.Path != "/json" {
		w.WriteHeader(404)
		return nil
	}
	
	data, err := ncrest.Marshal(entry, "json")
	if err != nil { return nil }
	
	if err := editJSONTmpl.Execute(w, string(data)); err != nil {
		w.WriteHeader(500)
	}
	
	return nil
}