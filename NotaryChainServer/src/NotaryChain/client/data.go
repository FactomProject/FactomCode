package main

import (
	"encoding/json"
	"io/ioutil"
	
	"NotaryChain/notarydata"
)

var entries []*notarydata.PlainEntry
var keys []*notarydata.ECDSAPrivKey

func init() {
	var blocks []*notarydata.Block
	
	source, err := ioutil.ReadFile("app/rest/store.json")
	if err != nil { panic(err) }
	
	err = json.Unmarshal(source, &blocks)
	if err != nil { panic(err) }
	
	entries = blocks[1].Entries
}

func getEntryCount() int {
	return len(entries)
}

func getEntry(id int) *notarydata.PlainEntry {
	return entries[id]
}

func getKeyCount() int {
	return len(keys)
}

func getKey(id int) *notarydata.ECDSAPrivKey {
	return keys[id]
}