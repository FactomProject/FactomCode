package main

import (
	"io/ioutil"
	
	"NotaryChain/notaryapi"
)

var entries []*notaryapi.Entry
var keys []*notaryapi.ECDSAPrivKey

func init() {
	data, err := ioutil.ReadFile("app/rest/store.1.block")
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

func getKey(id int) *notaryapi.ECDSAPrivKey {
	return keys[id]
}