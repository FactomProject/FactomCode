
package ldb

import (
	"bytes"
	"encoding/binary"
	"time"
//	"bytes"
	
//	"github.com/conformal/goleveldb/leveldb/iterator"
	"github.com/conformal/goleveldb/leveldb/util"

	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/conformal/goleveldb/leveldb"	
	"log"
)


// InsertEntry inserts an entry and put it in queue
func (db *LevelDb) InsertEntryAndQueue(entrySha *notaryapi.Hash, binaryEntry *[]byte, entry *notaryapi.Entry, chainID *[]byte) (err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()	
	
	var entryKey [] byte = []byte{byte(TBL_ENTRY)} 
	entryKey = append (entryKey, entrySha.Bytes ...)
	db.lbatch.Put(entryKey, *binaryEntry)	
	
	
	//EntryQueue format: Table Name (1 bytes) + Chain Type (4 bytes) + Timestamp (8 bytes) + Entry Hash (32 bytes)
	var key [] byte = []byte{byte(TBL_ENTRY_QUEUE)} 					// Table Name (1 bytes)
	key = append(key, *chainID ...) 									// Chain id (32 bytes)
	
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(entry.TimeStamp()))	
	key = append(key, binaryTimestamp ...) 								// Timestamp (8 bytes)
	
	key = append(key, entrySha.Bytes ...) 								// Entry Hash (32 bytes)
	
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
	
	var key [] byte = []byte{byte(TBL_ENTRY)} 
	key = append (key, entrySha.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	entry.UnmarshalBinary(data)
	
	return entry, nil
} 

// FetchEntriesFromQueue gets all of the ebentries that have not been processed
func (db *LevelDb) FetchEntriesFromQueue(chainID *[]byte, startTime *[]byte) (ebentries []*notaryapi.EBEntry, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey [] byte = []byte{byte(TBL_ENTRY_QUEUE)} 		  		// Table Name (1 bytes)
	fromkey = append(fromkey, *chainID ...) 							// Chain Type (32 bytes)
	fromkey = append(fromkey, *startTime ...) 							// Timestamp  (8 bytes)

	var tokey [] byte = []byte{byte(TBL_ENTRY_QUEUE)} 		  			// Table Name (4 bytes)
	tokey = append(tokey, *chainID ...) 								// Chain Type (4 bytes)
	
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(time.Now().Unix()))	
	tokey = append(tokey, binaryTimestamp ...) 							// Timestamp (8 bytes)
	
	ebEntrySlice := make([]*notaryapi.EBEntry, 0, 10) 	
	
	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	
	for iter.Next() {		
		if  bytes.Equal(iter.Value(), []byte{byte(STATUS_IN_QUEUE)}) {
			key := make([]byte, len(iter.Key()))			
			copy(key, iter.Key())
			ebEntry := new (notaryapi.EBEntry)
	
			ebEntry.SetTimeStamp(key[33:41])
			ebEntry.SetHash(key[41:73])
			ebEntrySlice = append(ebEntrySlice, ebEntry)		
		}

	}
	iter.Release()
	err = iter.Error()
	
	
	return ebEntrySlice, nil
}	

	
	