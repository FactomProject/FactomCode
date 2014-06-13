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

func unsignEntry(eid, sid int) notarydata.Signature {
	if eid < 0 || sid < 0 || eid >= getEntryCount() {
		return nil
	}
	
	entry := getEntry(eid)
	
	if sid >= len(entry.Signatures) {
		return nil
	}
	
	sig := entry.Signatures[sid]
	entry.Signatures = append(entry.Signatures[:sid], entry.Signatures[sid+1:]...)
	return sig
}