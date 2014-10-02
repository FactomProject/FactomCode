
package ldb

import (
	"github.com/FactomProject/FactomCode/notaryapi"	
	"github.com/conformal/goleveldb/leveldb"
	"errors"
	"log"
	"bytes"
	"encoding/binary"
	"time"

	"github.com/conformal/goleveldb/leveldb/util"
)


// FetchEBEntriesFromQueue gets all of the ebentries that have not been processed
func (db *LevelDb) FetchEBEntriesFromQueue(chainID *[]byte, startTime *[]byte) (ebentries []*notaryapi.EBEntry, err error) {
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


// ProcessEBlockBatche inserts the EBlock and update all it's ebentries in DB
func (db *LevelDb) ProcessEBlockBatch(eBlockHash *notaryapi.Hash, eblock *notaryapi.Block) error {

	if eblock !=  nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()
		
		if len(eblock.EBEntries) < 1 {
			return errors.New("Empty eblock!")
		}
				
		binaryEblock, err := eblock.MarshalBinary()
		if err != nil{
			return err
		}
		
		// Insert the binary entry block
		var key [] byte = []byte{byte(TBL_EB)} 
		key = append (key, eBlockHash.Bytes ...)			
		db.lbatch.Put(key, binaryEblock)
		
		// Insert the binary Entry Block process queue in order to create a FBEntry
		var qkey [] byte = []byte{byte(TBL_EB_QUEUE)} 									// Table Name (1 bytes)
		binaryTimestamp := make([]byte, 8)
		binary.BigEndian.PutUint64(binaryTimestamp, uint64(eblock.Header.TimeStamp))	
		qkey = append(qkey, binaryTimestamp ...) 										// Timestamp (8 bytes)	
		qkey = append(qkey, *eblock.Chain.ChainID ...) 									// Chain id (32 bytes)
		qkey = append (qkey, eBlockHash.Bytes ...)										// EBEntry Hash (32 bytes)
		db.lbatch.Put(qkey, []byte{byte(STATUS_IN_QUEUE)})
	
		// Update entry process queue for each entry in eblock
		for i:=0; i< len(eblock.EBEntries); i++  {
			var ebEntry notaryapi.EBEntry = *eblock.EBEntries[i] 
			var ebEntryKey [] byte = []byte{byte(TBL_ENTRY_QUEUE)} 		  				// Table Name (1 bytes)
			ebEntryKey = append(ebEntryKey, *(eblock.Chain.ChainID) ...) 				// Chain id (32 bytes)
			ebEntryKey = append(ebEntryKey, ebEntry.GetBinaryTimeStamp() ...) 			// Timestamp (8 bytes)
			ebEntryKey = append(ebEntryKey, ebEntry.Hash().Bytes ...) 					// Entry Hash (32 bytes)
			db.lbatch.Put(ebEntryKey, []byte{byte(STATUS_PROCESSED)})	
		}

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}