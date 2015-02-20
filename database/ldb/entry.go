package ldb

import (
	"encoding/binary"

	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"log"
	"strings"
	"time"
)

// InsertEntry inserts an entry and put it in queue
func (db *LevelDb) InsertEntryAndQueue(entrySha *notaryapi.Hash, binaryEntry *[]byte, entry *notaryapi.Entry, chainID *[]byte) (err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	var entryKey []byte = []byte{byte(TBL_ENTRY)}
	entryKey = append(entryKey, entrySha.Bytes...)
	db.lbatch.Put(entryKey, *binaryEntry)

	//EntryQueue format: Table Name (1 bytes) + Chain Type (4 bytes) + Timestamp (8 bytes) + Entry Hash (32 bytes)
	var key []byte = []byte{byte(TBL_ENTRY_QUEUE)} // Table Name (1 bytes)
	key = append(key, *chainID...)                 // Chain id (32 bytes)

	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(time.Now().Unix()))
	key = append(key, binaryTimestamp...) // Timestamp (8 bytes)

	key = append(key, entrySha.Bytes...) // Entry Hash (32 bytes)

	db.lbatch.Put(key, []byte{byte(STATUS_IN_QUEUE)})

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}

	return nil
}

// FetchEntry gets an entry by hash from the database.
func (db *LevelDb) FetchEntryByHash(entrySha *notaryapi.Hash) (entry *notaryapi.Entry, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_ENTRY)}
	key = append(key, entrySha.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		entry = new(notaryapi.Entry)
		entry.UnmarshalBinary(data)
	}
	return entry, nil
}

// FetchEntryInfoBranchByHash gets an EntryInfoBranch obj
func (db *LevelDb) FetchEntryInfoBranchByHash(entryHash *notaryapi.Hash) (entryInfoBranch *notaryapi.EntryInfoBranch, err error) {
	entryInfoBranch = new(notaryapi.EntryInfoBranch)
	entryInfoBranch.EntryHash = entryHash
	entryInfoBranch.EntryInfo, _ = db.FetchEntryInfoByHash(entryHash)

	if entryInfoBranch.EntryInfo != nil {
		entryInfoBranch.EBInfo, _ = db.FetchEBInfoByHash(entryInfoBranch.EntryInfo.EBHash)
	}

	if entryInfoBranch.EBInfo != nil {
		entryInfoBranch.DBBatch, _ = db.FetchDBBatchByHash(entryInfoBranch.EBInfo.DBHash)
	}

	return entryInfoBranch, nil
}

// FetchEntryInfoBranchByHash gets an EntryInfo obj
func (db *LevelDb) FetchEntryInfoByHash(entryHash *notaryapi.Hash) (entryInfo *notaryapi.EntryInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_ENTRY_INFO)}
	key = append(key, entryHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		entryInfo = new(notaryapi.EntryInfo)
		entryInfo.UnmarshalBinary(data)
	}
	return entryInfo, nil
}

// Initialize External ID map for explorer search
func (db *LevelDb) InitializeExternalIDMap() (extIDMap map[string]bool, err error) {

	var fromkey []byte = []byte{byte(TBL_ENTRY)} // Table Name (1 bytes)

	var tokey []byte = []byte{byte(TBL_ENTRY + 1)} // Table Name (1 bytes)

	extIDMap = make(map[string]bool)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		entry := new(notaryapi.Entry)
		entry.UnmarshalBinary(iter.Value())
		if entry.ExtIDs != nil {
			for i := 0; i < len(entry.ExtIDs); i++ {
				mapKey := string(iter.Key()[1:])
				mapKey = mapKey + strings.ToLower(string(entry.ExtIDs[i]))
				extIDMap[mapKey] = true
			}
		}

	}
	iter.Release()
	err = iter.Error()

	return extIDMap, nil
}
