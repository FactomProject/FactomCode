package main

import (
	"bytes"
	"fmt"
	"github.com/FactomProject/gobundle"
	"github.com/FactomProject/gocoding"
	"github.com/FactomProject/gocoding/json"
	"github.com/FactomProject/gocoding/html"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomapi"	
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sort"
)

type FlaggedEntry struct {
	Entry *ClientEntry
	Submitted *Submission
	
}

type Submission struct {
	Host string
	Confirmed int
	EntryHash string 
}

var entries map[int]*FlaggedEntry
var keys map[int]notaryapi.Key
var chains map[int]notaryapi.Chain

var marshaller gocoding.Marshaller
var marshallerHTML gocoding.Marshaller
var unmarshaller gocoding.Unmarshaller

var Settings = &struct {
	Server string
	NextChainID, NextKeyID, NextEntryID int
}{
	serverAddr,  
	0, 0, 0,
}

func init() {
	marshaller = json.NewMarshaller()
	marshallerHTML = html.NewMarshaller()
	unmarshaller = notaryapi.NewJSONUnmarshaller()
}



func EXPLODE(obj interface{}) error {
	panic(obj)
}


func loadStore() {
	keys = make(map[int]notaryapi.Key)
	
	chains = make(map[int]notaryapi.Chain)

	err := os.MkdirAll(gobundle.ConfigFile("store"), 0755)
	if err != nil { panic(err) }
	
	matches, err := filepath.Glob(gobundle.ConfigFile("store/entry*"))
	if err != nil { panic(err) }
	
	entries = make(map[int]*FlaggedEntry)
	
	exp := regexp.MustCompile(`.+store/entry(\d+)`)
	
	for _, match := range matches {
		sub := exp.FindStringSubmatch(match)
		if len(sub) != 2 { panic(fmt.Sprint("Bad entry file name: ", match)) }
		
		num, err := strconv.ParseInt(sub[1], 10, 64)
		if err != nil { panic(err) }
		
		data, err := ioutil.ReadFile(match)
		if err != nil { panic(err) }
		
		entry := new(FlaggedEntry)
		reader := gocoding.ReadBytes(data)
		err = factomapi.SafeUnmarshal(reader, entry)
		if err != nil { panic(err) }
		
		entries[int(num)] = entry
	}
}

func loadSettings() {
	data, err := ioutil.ReadFile(gobundle.ConfigFile("settings.json"))
	if err != nil { return }
	
	reader := gocoding.ReadBytes(data)
	err = json.Unmarshal(reader, &Settings)
	if err != nil { panic(err) }
}

func saveSettings() {
	file, err := os.OpenFile(gobundle.ConfigFile("settings.json"), os.O_CREATE | os.O_TRUNC | os.O_WRONLY, 0755)
	if err != nil { fmt.Fprintln(os.Stderr, err) }
	
	err = json.Marshal(file, Settings)
	if err != nil { fmt.Fprintln(os.Stderr, err) }
}

func getEntryCount() int {
	return len(entries)
}

func getActiveEntryIDs() []int {
	ids := make([]int, getEntryCount())
	i := 0
	for id, entry := range entries {
		if entry.Submitted != nil && entry.Submitted.Host != "" {
			continue
		}
		
		ids[i] = id
		i++
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids[:i])))
	return ids[:i]
}

func getPendingEntryIDs() []int {
	ids := make([]int, getEntryCount())
	i := 0
	for id, entry := range entries {
		if entry.Submitted == nil || entry.Submitted.Host == "" || entry.Submitted.Confirmed>0 {
			continue
		}
		
		ids[i] = id
		i++
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids[:i])))
	return ids[:i]
}

func getConfirmedEntryIDs() []int {
	ids := make([]int, getEntryCount())
	i := 0
	for id, entry := range entries {
		if entry.Submitted == nil || entry.Submitted.Host == "" || entry.Submitted.Confirmed ==0 {
			continue
		}
		
		ids[i] = id
		i++
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids[:i])))
	return ids[:i]
}

func RefreshPendingEntries(){
	ids := getPendingEntryIDs()
	
	for _, id := range ids {
		hash := new (notaryapi.Hash)	
		hash.Bytes, _ = notaryapi.DecodeBinary(&entries[id].Submitted.EntryHash)
		entryInfoBranch, _ := db.FetchEntryInfoBranchByHash(hash)
		if entryInfoBranch.FBBatch != nil {
			entries[id].Submitted.Confirmed =2
			storeEntry(id)
		} else if entryInfoBranch.EBInfo != nil{
			entries[id].Submitted.Confirmed =1
			storeEntry(id)			
		}
		
	}
}

func getEntry(id int) *ClientEntry {
	return entries[id].Entry
}

func getEntrySubmission(id int) *Submission {
	return entries[id].Submitted
}

func flagSubmitEntry(id int, entryHash string) {
	entries[id].Submitted = &Submission{Host: Settings.Server, EntryHash: entryHash}
	storeEntry(id)
}

func addEntry(entry *ClientEntry) (last int) {
	entries[Settings.NextEntryID] = &FlaggedEntry{Entry: entry}
	last = Settings.NextEntryID
	storeEntry(Settings.NextEntryID)
	Settings.NextEntryID++
	saveSettings()
	return
}

func storeEntry(id int) {
	buf := new(bytes.Buffer)
	
	entry := entries[id]
	if (entry == nil || entry.Entry == nil || entry.Entry.Data() == nil) {
		return
	}
	
	err := factomapi.SafeMarshal(buf, entry)
	if err != nil { fmt.Fprintln(os.Stderr, err); return }
	
	err = ioutil.WriteFile(gobundle.ConfigFile(fmt.Sprintf(`store/entry%d`, id)), buf.Bytes(), 0755)
	if err != nil { fmt.Fprintln(os.Stderr, err) }
}

func getKeyCount() int {
	return len(keys)
}
 
func getChainCount() int {
	return len(chains)
}

func getKeyIDs() []int {
	ids := make([]int, getKeyCount())
	i := 0
	for id, _ := range keys {
		ids[i] = id
		i++
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids[:i])))
	return ids[:i]
}
func getChainIDs() []int {
	ids := make([]int, getChainCount())
	i := 0
	for id, _ := range chains {
		ids[i] = id
		i++
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids[:i])))
	return ids[:i]
}

func getKey(id int) notaryapi.Key {
	return keys[id]
}

func getChain(id int) notaryapi.Chain {
	return chains[id]
}

func addKey(key notaryapi.Key) {
	keys[Settings.NextKeyID] = key
	storeKey(Settings.NextKeyID)
	Settings.NextKeyID++
}

func addChain(chain notaryapi.Chain) {
	chains[Settings.NextChainID] = chain
	storeKey(Settings.NextChainID)
	Settings.NextChainID++
}

func storeKey(id int) {
	
}
/*
func getEBlock() (eBlock *notaryapi.Block){
	
	eBlock = new (notaryapi.Block)
	return eBlock
	
}
*/

