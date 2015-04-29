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
