package ldb

import (
	"github.com/FactomProject/FactomCode/notaryapi"		
	"github.com/conformal/goleveldb/leveldb"
	"errors"
	"log"
	"github.com/conformal/goleveldb/leveldb/util"
	"bytes"
	"time"
	"encoding/binary"	
)


// FetchFBEntriesFromQueue gets all of the fbentries that have not been processed
func (db *LevelDb) FetchFBEntriesFromQueue(startTime *[]byte) (fbentries []*notaryapi.FBEntry, err error){	
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey [] byte = []byte{byte(TBL_EB_QUEUE)} 		  			// Table Name (1 bytes)
	fromkey = append(fromkey, *startTime ...) 							// Timestamp  (8 bytes)

	var tokey [] byte = []byte{byte(TBL_EB_QUEUE)} 		  				// Table Name (4 bytes)
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(time.Now().Unix()))	
	tokey = append(tokey, binaryTimestamp ...) 							// Timestamp  (8 bytes)
	
	fbEntrySlice := make([]*notaryapi.FBEntry, 0, 10) 	
	
	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	
	for iter.Next() {		
		if  bytes.Equal(iter.Value(), []byte{byte(STATUS_IN_QUEUE)}) {
			key := make([]byte, len(iter.Key()))			
			copy(key, iter.Key())
			fbEntry := new (notaryapi.FBEntry)
	
			fbEntry.SetTimeStamp(key[1:9])							// Timestamp (8 bytes)
			cid := key[9:41]				
			fbEntry.ChainID = &cid 									// Chain id (32 bytes)
			fbEntry.SetHash(key[41:73])								// Entry Hash (32 bytes)

			fbEntrySlice = append(fbEntrySlice, fbEntry)		
		}

	}
	iter.Release()
	err = iter.Error()
	
	return fbEntrySlice, nil
}	
// ProcessFBlockBatche inserts the FBlock and update all it's fbentries in DB
func (db *LevelDb) ProcessFBlockBatch(fBlockHash *notaryapi.Hash, fblock *notaryapi.FBlock) error {

	if fblock !=  nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()
		
		if len(fblock.FBEntries) < 1 {
			return errors.New("Empty fblock!")
		}
				
		binaryFblock, err := fblock.MarshalBinary()
		if err != nil{
			return err
		}
		
		// Insert the binary factom block
		var key [] byte = []byte{byte(TBL_FB)} 
		key = append (key, fBlockHash.Bytes ...)			
		db.lbatch.Put(key, binaryFblock)
		
		// Update FBEntry process queue for each fbEntry in fblock
		for i:=0; i< len(fblock.FBEntries); i++  {
			var fbEntry notaryapi.FBEntry = *fblock.FBEntries[i] 
			var fbEntryKey [] byte = []byte{byte(TBL_EB_QUEUE)} 		  			// Table Name (1 bytes)
			fbEntryKey = append(fbEntryKey, fbEntry.GetBinaryTimeStamp() ...) 		// Timestamp (8 bytes)
			fbEntryKey = append(fbEntryKey, *fbEntry.ChainID ...) 					// Chain id (32 bytes)
			fbEntryKey = append(fbEntryKey, fbEntry.Hash().Bytes ...) 				// Entry Hash (32 bytes)
			db.lbatch.Put(fbEntryKey, []byte{byte(STATUS_PROCESSED)})
		 
		}

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

