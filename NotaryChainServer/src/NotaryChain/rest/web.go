package main 

import (
	"flag"
	"net/http"
	"net/url"
	ncdata "NotaryChain/data"
	"encoding/json"
	"strconv"
	"strings"
	"fmt"
	"errors"
)

var portNumber *int = flag.Int("p", 8083, "Set the port to listen on")

var blocks []*ncdata.Block

func load() {
	source := `[
		{
			"blockID": 0,
			"previousHash": null,
			"entries": [],
			"salt": {
				"bytes": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
			}
		},
		{
			"blockID": 1,
			"previousHash": {
				"bytes": "LDTOHfI7g4xavyp/ZDfMo9MGftUJ/yXxHfaxG1grUes="
			},
			"entries": [
				{
					"entryType": 0,
					"structuredData": "EBESEw==",
					"signatures": [],
					"timeSamp": 0
				}
			],
			"salt": {
				"bytes": "HJ7OyQ4o0kYWUEGGNYeKXJHkn0dYbs918rDLuU6JcRI="
			}
		}
	]`
	
	if err := json.Unmarshal([]byte(source), &blocks); err != nil {
		panic(err)
	}
}

func main() {
	load()
	
	http.HandleFunc("/", ServeRESTfulHTTP)
	http.ListenAndServe(":" + strconv.Itoa(*portNumber), nil)
}

func ServeRESTfulHTTP(w http.ResponseWriter, r *http.Request) {
	path, _, accept, _, _ := Parse(r)
	
	fmt.Println(accept)
	
	resource, _ := Find(path)
	s, _ := json.Marshal(resource)
	w.Write(s)
}

func Parse(r *http.Request) (path []string, method string, accept string, form url.Values, err error) {
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
	
	if len(path[0]) == 0 {
		path = path[1:]
	}
	
	if len(path[len(path) - 1]) == 0 {
		path = path[:len(path) - 1]
	}
	
	method = r.Method
	
	for _, accept = range r.Header["Accept"] {
		accept, err = ParseAccept(accept, bits)
		if err == nil {
			break
		}
	}
	
	if err != nil {
		return
	}
	
	form = r.Form
	
	return
}

func ParseAccept(accept string, ext []string) (string, error) {
	switch accept {
	case "text/plain":
		if len(ext) == 1 && ext[0] != "txt" {
			return ext[0], nil
		}
		return "text", nil
		
	case "application/json":
		return "json", nil
		
	case "application/xml", "text/xml":
		return "xml", nil
		
	case "text/html":
		return "html", nil
	}
	
	return "", errors.New("406")
}

func Find(path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, nil // missing version spec
	}
	
	ver, path := path[0], path[1:] // capture version spec
	
	if !strings.HasPrefix(ver, "v") {
		return nil, nil // malformated version spec
	}
	
	ver = strings.TrimPrefix(ver, "v")
	
	if ver == "1" {
		return FindV1(path)
	}
	
	return nil, nil // bad version spec
}

func FindV1(path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, nil // no request
	}
	
	root, path := path[0], path[1:] // capture root spec
	
	if strings.ToLower(root) != "blocks" {
		return nil, nil // bad root spec
	}
	
	return FindV1InBlocks(path, blocks)
}

func FindV1InBlocks(path []string, blocks []*ncdata.Block) (interface{}, error) {
	if len(path) == 0 {
		return blocks, nil
	}
	
	// capture root spec
	_id, err := strconv.Atoi(path[0])
	id := uint64(_id)
	path = path[1:]
	
	if err != nil {
		return nil, err // some other error
	}
	
	if len(blocks) == 0 {
		return nil, nil // 404
	}
	
	idOffset := blocks[0].BlockID
	
	if id < idOffset {
		return nil, nil // 404 
	}
	
	id = id - idOffset
	
	if len(blocks) <= int(id) {
		return nil, nil // 404
	}
	
	return FindV1InBlock(path, blocks[id])
}

func FindV1InBlock(path []string, block *ncdata.Block) (interface{}, error) {
	if len(path) == 0 {
		return block, nil
	}
	
	root, path := path[0], path[1:] // capture root spec
	
	if strings.ToLower(root) != "entries" {
		return nil, nil // bad root spec
	}
	
	return FindV1InEntries(path, block.Entries)
}

func FindV1InEntries(path []string, entries []*ncdata.PlainEntry) (interface{}, error) {
	if len(path) == 0 {
		return entries, nil
	}
	
	// capture root spec
	id, err := strconv.Atoi(path[0])
	path = path[1:]
	
	if err != nil {
		return nil, err // some other error
	}
	
	if len(entries) <= id {
		return nil, nil // 404
	}
	
	return entries[id], nil
}








