package main

import (
	"encoding/json"
	"io/ioutil"
	
	"NotaryChain/notaryapi"
)

var entries []*notaryapi.Entry
var keys []*notaryapi.ECDSAPrivKey

func init() {
	var blocks []*notaryapi.Block
	
	source, err := ioutil.ReadFile("app/rest/store.json")
	if err != nil { panic(err) }
	
	err = json.Unmarshal(source, &blocks)
	if err != nil { panic(err) }
	 
	entries = blocks[1].Entries
}

func getEntryCount() int {
	return len(entries)
}

func getEntry(id int) *notaryapi.Entry {
	return entries[id]
}

func getKeyCount() int {
	return len(keys)
}

func getKey(id int) *notaryapi.ECDSAPrivKey {
	return keys[id]
}