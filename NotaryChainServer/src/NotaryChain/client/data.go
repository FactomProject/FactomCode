package main

import (
	"encoding/json"
	"io/ioutil"
	
	"NotaryChain/notaryapi"
)

var entries []*notaryapi.DataEntry
var keys []*notaryapi.ECDSAPrivKey

func init() {
	var blocks []*notaryapi.Block
	
	source, err := ioutil.ReadFile("app/rest/store.json")
	if err != nil { panic(err) }
	
	err = json.Unmarshal(source, &blocks)
	if err != nil { panic(err) }
	 
	entries = []*notaryapi.DataEntry{blocks[1].Entries[0].(*notaryapi.DataEntry)}
}

func getEntryCount() int {
	return len(entries)
}

func getEntry(id int) *notaryapi.DataEntry {
	return entries[id]
}

func getKeyCount() int {
	return len(keys)
}

func getKey(id int) *notaryapi.ECDSAPrivKey {
	return keys[id]
}