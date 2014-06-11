package main 

import (
	"errors"
	"flag"
	"fmt"
	"reflect"
	"time"
	"strconv"
	"strings"
	"sync"
	
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
	
	"github.com/firelizzard18/dynrsrc"
	
	"NotaryChain/notarydata"
	"NotaryChain/restapi"
)

var portNumber *int = flag.Int("p", 8083, "Set the port to listen on")
var blocks []*notarydata.Block
var blockMutex = &sync.Mutex{}
var tickers [2]*time.Ticker

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init() {
	dynrsrc.Start(watchError, readError)
	restapi.StartDynamic(readError)
	
	source, err := ioutil.ReadFile("app/rest/store.json")
	if err != nil { panic(err) }
	
	err = json.Unmarshal(source, &blocks)
	if err != nil { panic(err) }
	
	for i := 0; i < len(blocks); i = i + 1 {
		if uint64(i) != blocks[i].BlockID {
			panic(errors.New("BlockID does not equal index"))
		}
	}
	notarydata.UpdateNextBlockID(uint64(len(blocks)))
	
	tickers[0] = time.NewTicker(time.Minute * 5)
	tickers[1] = time.NewTicker(time.Hour)
	
	go func() {
		for _ = range tickers[0].C {
			notarize()
		}
	}()
	
	go func() {
		for _ = range tickers[1].C {
			save()
		}
	}()
}

func main() {
	defer func() {
		tickers[0].Stop()
		tickers[1].Stop()
		save()
		dynrsrc.Stop()
	}()
	
	http.HandleFunc("/", serveRESTfulHTTP)
	http.ListenAndServe(":" + strconv.Itoa(*portNumber), nil)
}

func notarize() {
	fmt.Println("Checking if should send current block")
	blockMutex.Lock()
	fmt.Println("Sending block, creating new block")
	blockMutex.Unlock()
}

func save() {
	blockMutex.Lock()
	data, err := json.Marshal(blocks)
	blockMutex.Unlock()
	if err != nil { panic(err) }
	
	err = ioutil.WriteFile("app/rest/store.json", data, 0644)
	if err != nil { panic(err) }
}

func serveRESTfulHTTP(w http.ResponseWriter, r *http.Request) {
	var resource interface{}
	var data []byte
	var err *restapi.Error
	
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
			var r *restapi.Error
			
			data, r = restapi.Marshal(err, accept)
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
			err = restapi.CreateError(restapi.ErrorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}
		
		resource, err = post("/" + strings.Join(path, "/"), form)
		
	default:
		err = restapi.CreateError(restapi.ErrorBadMethod, fmt.Sprintf(`The HTTP %s method is not supported`, method))
		return
	}
	
	data, err = restapi.Marshal(resource, accept)
}

var blockPtrType = reflect.TypeOf((*notarydata.Block)(nil)).Elem()

func post(context string, form url.Values) (interface{}, *restapi.Error) {
	newEntry := new(notarydata.PlainEntry)
	format, data := form.Get("format"), form.Get("data")
	
	switch format {
	case "", "json":
		err := json.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, restapi.CreateError(restapi.ErrorJSONUnmarshal, err.Error())
		}
		
	case "xml":
		err := xml.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, restapi.CreateError(restapi.ErrorXMLUnmarshal, err.Error())
		}
	
	default:
		return nil, restapi.CreateError(restapi.ErrorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}
	
	if newEntry == nil {
		return nil, restapi.CreateError(restapi.ErrorInternal, `Entity to be POSTed is nil`)
	}
	
	newEntry.TimeStamp = time.Now().Unix()
	
	blockMutex.Lock()
	err := blocks[len(blocks) - 1].AddEntry(newEntry)
	blockMutex.Unlock()
	
	if err != nil {
		return nil, restapi.CreateError(restapi.ErrorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}
	
	return newEntry, nil
}
