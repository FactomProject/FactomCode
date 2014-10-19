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
			fbEntry.ChainID = new (notaryapi.Hash)
			fbEntry.ChainID.Bytes = cid								// Chain id (32 bytes)
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
			fbEntryKey = append(fbEntryKey, fbEntry.ChainID.Bytes ...) 				// Chain id (32 bytes)
			fbEntryKey = append(fbEntryKey, fbEntry.Hash().Bytes ...) 				// Entry Hash (32 bytes)
			db.lbatch.Put(fbEntryKey, []byte{byte(STATUS_PROCESSED)})
			
			if isLookupDB {
				// Create an EBInfo and insert it into db
				var ebInfo = new (notaryapi.EBInfo)
				ebInfo.EBHash = fbEntry.Hash()
				ebInfo.FBHash = fBlockHash
				ebInfo.FBBlockNum = fblock.Header.BlockID
				ebInfo.ChainID = fbEntry.ChainID
			 	var ebInfoKey [] byte = []byte{byte(TBL_EB_INFO)} 
			 	ebInfoKey = append(ebInfoKey, ebInfo.EBHash.Bytes ...) 
			 	binaryEbInfo, _ := ebInfo.MarshalBinary()
				db.lbatch.Put(ebInfoKey, binaryEbInfo)	
			}	 	
		}
		
		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}


// Insert the Factom Block meta data into db
func (db *LevelDb)InsertFBInfo(fbHash *notaryapi.Hash, fbInfo *notaryapi.FBInfo) (err error){
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()	
	
	// Insert FBInfo
	var Key [] byte = []byte{byte(TBL_FB_INFO)} 
	Key = append (Key, fbHash.Bytes ...)
	
	binaryEBInfo, _ := fbInfo.MarshalBinary()
	db.lbatch.Put(Key, binaryEBInfo)	

	// Create a Factom Block Number cross reference
	var fbNumKey [] byte = []byte{byte(TBL_FB_NUM)} 
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, fbInfo.FBlockID)		
	fbNumKey = append (fbNumKey, buf.Bytes() ...)
	
	binaryHash, _ := fbInfo.FBHash.MarshalBinary()
	db.lbatch.Put(fbNumKey, binaryHash)
	
	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}	

	return nil
} 


// FetchFBInfoByHash gets an FBInfo obj
func (db *LevelDb) FetchFBInfoByHash(fbHash *notaryapi.Hash) (fbInfo *notaryapi.FBInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	var key [] byte = []byte{byte(TBL_FB_INFO)} 
	key = append (key, fbHash.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	if data != nil{
		fbInfo = new (notaryapi.FBInfo)
		fbInfo.UnmarshalBinary(data)
	}
	
	return fbInfo, nil
} 

// FetchFBlock gets an entry by hash from the database.
func (db *LevelDb) FetchFBlockByHash(fBlockHash *notaryapi.Hash) (fBlock *notaryapi.FBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	var key [] byte = []byte{byte(TBL_FB)} 
	key = append (key, fBlockHash.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	if data != nil{
		fBlock = new (notaryapi.FBlock)
		fBlock.UnmarshalBinary(data)
	}
	
	return fBlock, nil
} 

// FetchAllFBInfo gets all of the fbInfo 
func (db *LevelDb) FetchAllFBInfos() (fbInfos []notaryapi.FBInfo, err error){	
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey [] byte = []byte{byte(TBL_FB_INFO)} 		  			// Table Name (1 bytes)						// Timestamp  (8 bytes)

	var tokey [] byte 		  											// Table Name (1 bytes)

	
	fbInfoSlice := make([]notaryapi.FBInfo, 0, 10) 	
	
	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	
	for iter.Next() {		
		var fbInfo notaryapi.FBInfo
		fbInfo.UnmarshalBinary(iter.Value())
		
		fbInfoSlice = append(fbInfoSlice, fbInfo)		

	}
	iter.Release()
	err = iter.Error()
	
	return fbInfoSlice, nil
}	

// FetchFBInfoByHash gets an FBInfo obj
func (db *LevelDb) FetchAllDBRecordsByFBHash(fbHash *notaryapi.Hash) (ldbMap map[string]string, err error) {
//	db.dbLock.Lock()
//	defer db.dbLock.Unlock()
	if ldbMap == nil {
		ldbMap = make(map[string]string)
	}
	
	var fblock notaryapi.FBlock
	var eblock notaryapi.Block
	
	//FBlock
	var key [] byte = []byte{byte(TBL_FB)} 
	key = append (key, fbHash.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
		
	
	if data == nil {
		return nil, errors.New("FBlock not found for FBHash: " + fbHash.String())
	} else {
		fblock.UnmarshalBinary(data)
		ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
	}
	
	f, _:=db.FetchEBInfoByHash(fbHash)
	if f==nil{
		log.Println("f is null")
	}
	//FBInfo
	key = []byte{byte(TBL_FB_INFO)} 
	key = append (key, fbHash.Bytes ...)	
	data, err = db.lDb.Get(key, db.ro) 
	
	if data == nil {
		return nil, errors.New("FBInfo not found for FBHash: " + fbHash.String())
	} else {
		ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
	}
	
	//EBlocks
	for _, fbentry := range fblock.FBEntries{
		//EBlock
		key = []byte{byte(TBL_EB)}
		key = append (key, fbentry.Hash().Bytes ...)	
		data, err = db.lDb.Get(key, db.ro)
		if data == nil {
			return nil, errors.New("EBlock not found for EBHash: " + fbentry.Hash().String())
		} else {
			eblock.UnmarshalBinary(data)
			ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
		}		
		//EBInfo
		key = []byte{byte(TBL_EB_INFO)}
		key = append (key, fbentry.Hash().Bytes ...)	
		data, err = db.lDb.Get(key, db.ro)
		if data == nil {
			return nil, errors.New("EBInfo not found for EBHash: " + fbentry.Hash().String())
		} else {
			ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
		}		
		
		//Entries
		for _, ebentry := range eblock.EBEntries{
			//Entry
			key = []byte{byte(TBL_ENTRY)}
			key = append (key, ebentry.Hash().Bytes ...)	
			data, err = db.lDb.Get(key, db.ro)
			if data == nil {
				return nil, errors.New("Entry not found for entry hash: " + ebentry.Hash().String())
			} else {
				ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
			}		
			//EntryInfo
			key = []byte{byte(TBL_ENTRY_INFO)}
			key = append (key, ebentry.Hash().Bytes ...)	
			data, err = db.lDb.Get(key, db.ro)
			if data == nil {
				return nil, errors.New("EntryInfo not found for entry hash: " + ebentry.Hash().String())
			} else {
				ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
			}	
		}			
		
	}
	
	
	
	return ldbMap, nil
} 

// InsertAllDBRecords inserts all key value pairs from map into db
func (db *LevelDb) InsertAllDBRecords(ldbMap map[string]string) (err error){
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()	
	
    for key, value := range ldbMap{
    	binaryKey, _ := notaryapi.DecodeBinary(&key)
    	banaryValue, _ := notaryapi.DecodeBinary(&value)
		db.lbatch.Put(binaryKey, banaryValue)	    
    }
	
	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}	

	return nil	
}