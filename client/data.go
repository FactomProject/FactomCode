package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/firelizzard18/gobundle"
	"github.com/firelizzard18/gocoding"
	"github.com/firelizzard18/gocoding/json"
	"github.com/firelizzard18/gocoding/html"
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type FlaggedEntry struct {
	Entry *notaryapi.Entry
	Submitted *Submission
}

type Submission struct {
	Host string
	Confirmed bool
}

var entries map[int]*FlaggedEntry
var keys map[int]notaryapi.Key

var marshaller gocoding.Marshaller
var marshallerHTML gocoding.Marshaller
var unmarshaller gocoding.Unmarshaller

var Settings = &struct {
	Server string
	NextKeyID, NextEntryID int
}{
	"localhost:8083",
	0, 0,
}

func init() {
	marshaller = json.NewMarshaller()
	marshallerHTML = html.NewMarshaller()
	unmarshaller = notaryapi.NewJSONUnmarshaller()
}

func safeRecover(obj interface{}) error {
	if err, ok := obj.(error); ok {
		return err
	}
	
	return errors.New(fmt.Sprint(obj))
}

func EXPLODE(obj interface{}) error {
	panic(obj)
}

func safeMarshal(writer io.Writer, obj interface{}) error {
	renderer := json.Render(writer)
	renderer.SetRecoverHandler(safeRecover)
	return marshaller.Marshal(renderer, obj)
}

func safeMarshalHTML(writer io.Writer, obj interface{}) error {
	renderer := html.Render(writer)
	renderer.SetRecoverHandler(safeRecover)
	return marshallerHTML.Marshal(renderer, obj)
}

func safeUnmarshal(reader gocoding.SliceableRuneReader, obj interface{}) error {
	scanner := json.Scan(reader)
	scanner.SetRecoverHandler(safeRecover)
	return unmarshaller.Unmarshal(scanner, obj)
}

func loadStore() {
	keys = make(map[int]notaryapi.Key)

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
		err = safeUnmarshal(reader, entry)
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
	return ids[:i]
}

func getEntry(id int) *notaryapi.Entry {
	return entries[id].Entry
}

func getEntrySubmission(id int) *Submission {
	return entries[id].Submitted
}

func flagSubmitEntry(id int) {
	entries[id].Submitted = &Submission{Host: Settings.Server}
	storeEntry(id)
}

func addEntry(entry *notaryapi.Entry) (last int) {
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
	if entry == nil { return }
	
	err := safeMarshal(buf, entry)
	if err != nil { fmt.Fprintln(os.Stderr, err); return }
	
	err = ioutil.WriteFile(gobundle.ConfigFile(fmt.Sprintf(`store/entry%d`, id)), buf.Bytes(), 0755)
	if err != nil { fmt.Fprintln(os.Stderr, err) }
}

func getKeyCount() int {
	return len(keys)
}

func getKeyIDs() []int {
	ids := make([]int, getKeyCount())
	i := 0
	for id, _ := range keys {
		ids[i] = id
		i++
	}
	return ids[:i]
}

func getKey(id int) notaryapi.Key {
	return keys[id]
}

func addKey(key notaryapi.Key) {
	keys[Settings.NextKeyID] = key
	storeKey(Settings.NextKeyID)
	Settings.NextKeyID++
}

func storeKey(id int) {
	
}