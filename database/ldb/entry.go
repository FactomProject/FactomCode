package ldb

import (

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"log"
	"strings"
)

// InsertEntry inserts an entry
func (db *LevelDb) InsertEntry(entrySha *common.Hash, binaryEntry *[]byte, entry *common.Entry, chainID *[]byte) (err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	var entryKey []byte = []byte{byte(TBL_ENTRY)}
	entryKey = append(entryKey, entrySha.Bytes...)
	db.lbatch.Put(entryKey, *binaryEntry)

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}

	return nil
}

// FetchEntry gets an entry by hash from the database.
func (db *LevelDb) FetchEntryByHash(entrySha *common.Hash) (entry *common.Entry, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_ENTRY)}
	key = append(key, entrySha.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		entry = new(common.Entry)
		entry.UnmarshalBinary(data)
	}
	return entry, nil
}

// FetchEntryInfoBranchByHash gets an EntryInfoBranch obj
func (db *LevelDb) FetchEntryInfoBranchByHash(entryHash *common.Hash) (entryInfoBranch *common.EntryInfoBranch, err error) {
	entryInfoBranch = new(common.EntryInfoBranch)
	entryInfoBranch.EntryHash = entryHash
	entryInfoBranch.EntryInfo, _ = db.FetchEntryInfoByHash(entryHash)

	if entryInfoBranch.EntryInfo != nil {
		entryInfoBranch.EBInfo, _ = db.FetchEBInfoByHash(entryInfoBranch.EntryInfo.EBHash)
	}

	if entryInfoBranch.EBInfo != nil {
		entryInfoBranch.DBInfo, _ = db.FetchDBInfoByHash(entryInfoBranch.EBInfo.DBHash)
	} 

	return entryInfoBranch, nil
}

// FetchEntryInfoBranchByHash gets an EntryInfo obj
func (db *LevelDb) FetchEntryInfoByHash(entryHash *common.Hash) (entryInfo *common.EntryInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_ENTRY_INFO)}
	key = append(key, entryHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		entryInfo = new(common.EntryInfo)
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
		entry := new(common.Entry)
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
