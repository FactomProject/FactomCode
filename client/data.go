package main

import (
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"github.com/firelizzard18/gobundle"
	"io/ioutil"
)

var entries []*notaryapi.Entry
var keys []notaryapi.Key

var Settings = struct {Server string} {
	"localhost:8083",
}

func load() {
	data, err := ioutil.ReadFile(gobundle.DataFile("store.1.block"))
	if err != nil { panic(err) }
	
	block := new(notaryapi.Block)
	err = block.UnmarshalBinary(data)
	if err != nil { panic(err) }
	 
	entries = block.Entries
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

func getKey(id int) notaryapi.Key {
	return keys[id]
}

func addKey(key notaryapi.Key) {
	keys = append(keys, key)
}