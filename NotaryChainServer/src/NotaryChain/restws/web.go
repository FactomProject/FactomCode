package main 

import (
	"flag"
	"net/http"
	"net/url"
	ncdata "NotaryChain/data"
	ncrest "NotaryChain/rest"
	"encoding/json"
	"encoding/xml"
	"strconv"
	"fmt"
	"strings"
	"reflect"
	"time"
	"errors"
	"sync"
	"io/ioutil"
)

var portNumber *int = flag.Int("p", 8083, "Set the port to listen on")
var blocks []*ncdata.Block
var blockMutex = &sync.Mutex{}

func init() {
	source, err := ioutil.ReadFile("test/rest/store.json")
	if err != nil { panic(err) }
	
	if err := json.Unmarshal(source, &blocks); err != nil { panic(err) }
	
	for i := 0; i < len(blocks); i = i + 1 {
		if uint64(i) != blocks[i].BlockID {
			panic(errors.New("BlockID does not equal index"))
		}
	}
	ncdata.UpdateNextBlockID(uint64(len(blocks)))
	
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

func main() {
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
	var err *ncrest.Error
	
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
			var r *ncrest.Error
			
			data, r = ncrest.Marshal(err, accept)
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
			err = ncrest.CreateError(ncrest.ErrorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}
		
		resource, err = post("/" + strings.Join(path, "/"), form)
		
	default:
		err = ncrest.CreateError(ncrest.ErrorBadMethod, fmt.Sprintf(`The HTTP %s method is not supported`, method))
		return
	}
	
	data, err = ncrest.Marshal(resource, accept)
}

var blockPtrType = reflect.TypeOf((*ncdata.Block)(nil)).Elem()

func post(context string, form url.Values) (interface{}, *ncrest.Error) {
	newEntry := new(ncdata.PlainEntry)
	format, data := form.Get("format"), form.Get("data")
	
	switch format {
	case "", "json":
		err := json.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, ncrest.CreateError(ncrest.ErrorJSONUnmarshal, err.Error())
		}
		
	case "xml":
		err := xml.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, ncrest.CreateError(ncrest.ErrorXMLUnmarshal, err.Error())
		}
	
	default:
		return nil, ncrest.CreateError(ncrest.ErrorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}
	
	if newEntry == nil {
		return nil, ncrest.CreateError(ncrest.ErrorInternal, `Entity to be POSTed is nil`)
	}
	
	newEntry.TimeStamp = time.Now().Unix()
	
	blockMutex.Lock()
	err := blocks[len(blocks) - 1].AddEntry(newEntry)
	blockMutex.Unlock()
	
	if err != nil {
		return nil, ncrest.CreateError(ncrest.ErrorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}
	
	return newEntry, nil
}
