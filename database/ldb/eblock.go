
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
		qkey = append(qkey, eblock.Chain.ChainID.Bytes ...) 									// Chain id (32 bytes)
		qkey = append (qkey, eBlockHash.Bytes ...)										// EBEntry Hash (32 bytes)
		db.lbatch.Put(qkey, []byte{byte(STATUS_IN_QUEUE)})
	
		// Update entry process queue for each entry in eblock
		for i:=0; i< len(eblock.EBEntries); i++  {
			var ebEntry notaryapi.EBEntry = *eblock.EBEntries[i] 
			var ebEntryKey [] byte = []byte{byte(TBL_ENTRY_QUEUE)} 		  				// Table Name (1 bytes)
			ebEntryKey = append(ebEntryKey, eblock.Chain.ChainID.Bytes ...) 				// Chain id (32 bytes)
			ebEntryKey = append(ebEntryKey, ebEntry.GetBinaryTimeStamp() ...) 			// Timestamp (8 bytes)
			ebEntryKey = append(ebEntryKey, ebEntry.Hash().Bytes ...) 					// Entry Hash (32 bytes)
			db.lbatch.Put(ebEntryKey, []byte{byte(STATUS_PROCESSED)})	
			
			if isLookupDB {
				// Create an EntryInfo and insert it into db
				var entryInfo = new (notaryapi.EntryInfo)
				entryInfo.EntryHash = ebEntry.Hash()
				entryInfo.EBHash = eBlockHash
				entryInfo.EBBlockNum = eblock.Header.BlockID
			 	var entryInfoKey [] byte = []byte{byte(TBL_ENTRY_INFO)} 
			 	entryInfoKey = append(entryInfoKey, entryInfo.EntryHash.Bytes ...) 
			 	binaryEntryInfo, _ := entryInfo.MarshalBinary()
				db.lbatch.Put(entryInfoKey, binaryEntryInfo)	
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

// FetchEBInfoByHash gets an EBInfo obj
func (db *LevelDb) FetchEBInfoByHash(ebHash *notaryapi.Hash) (ebInfo *notaryapi.EBInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	var key [] byte = []byte{byte(TBL_EB_INFO)} 
	key = append (key, ebHash.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	if data != nil{
		ebInfo = new (notaryapi.EBInfo)
		ebInfo.UnmarshalBinary(data)
	}
	
	return ebInfo, nil
} 


// FetchEntryBlock gets an entry by hash from the database.
func (db *LevelDb) FetchEBlockByHash(eBlockHash *notaryapi.Hash) (eBlock *notaryapi.Block, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	var key [] byte = []byte{byte(TBL_EB)} 
	key = append (key, eBlockHash.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	if data != nil{
		eBlock = new (notaryapi.Block)
		eBlock.UnmarshalBinary(data)
	}
	return eBlock, nil
} 

// InsertChain inserts the newly created chain into db
func (db *LevelDb)	InsertChain(chain *notaryapi.Chain) (err error){
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()	
	
	binaryChain, _ := chain.MarshalBinary()
	
	var chainByHashKey [] byte = []byte{byte(TBL_CHAIN_HASH)} 
	chainByHashKey = append (chainByHashKey, chain.ChainID.Bytes ...)
	
	db.lbatch.Put(chainByHashKey, binaryChain)	
	
	var chainByNameKey [] byte = []byte{byte(TBL_CHAIN_NAME)} 
	chainByNameKey = append (chainByNameKey, []byte(notaryapi.EncodeChainNameToString(chain.Name)) ...)
	
	// Cross reference to chain id
	db.lbatch.Put(chainByNameKey, chain.ChainID.Bytes)	
	
	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}	

	return nil
}	

// FetchChainByHash gets a chain by chainID
func (db *LevelDb) 	FetchChainByHash(chainID *notaryapi.Hash) (chain *notaryapi.Chain, err error){
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	var key [] byte = []byte{byte(TBL_CHAIN_HASH)} 
	key = append (key, chainID.Bytes ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	if data != nil{
		chain = new (notaryapi.Chain)
		chain.UnmarshalBinary(data)
	}
	return chain, nil
}	

// FetchChainIDByName gets a chainID by chain name
func (db *LevelDb) 	FetchChainIDByName(chainName [][]byte) (chainID *notaryapi.Hash, err error){
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	name := notaryapi.EncodeChainNameToString(chainName)
	
	var key [] byte = []byte{byte(TBL_CHAIN_NAME)} 
	key = append (key, []byte(name) ...)	
	data, err := db.lDb.Get(key, db.ro)
	
	if data != nil{
		chainID = new (notaryapi.Hash)
		chainID.Bytes = make([]byte, len(data))
		copy(chainID.Bytes, data)
	}
	return chainID, nil
}
// FetchChainByName gets a chain by chain name
func (db *LevelDb) 	FetchChainByName(chainName [][]byte) (chain *notaryapi.Chain, err error){
	
	
	chainID,_ := db.FetchChainIDByName(chainName)
	
	if chainID != nil{
		chain, _ = db.FetchChainByHash(chainID)
	}
	return chain, nil		
	
}		

// FetchAllChainByName gets all of the chains under the path - name
func (db *LevelDb)	FetchAllChainsByName(chainName [][]byte) (chains *[]notaryapi.Chain, err error)	{

	chainSlice := make([]notaryapi.Chain, 0, 10) 	
	
	chainIDSlice,_ := db.FetchAllChainIDsByName(chainName)
	
	for _, chainID := range *chainIDSlice{			
		chain,_ := db.FetchChainByHash (&chainID)
		if chain != nil{
			chainSlice = append(chainSlice, *chain)
		}				
	}
	
	return &chainSlice, nil	
}
// FetchAllChainIDsByName gets all of the chainIDs under the path - name
func (db *LevelDb)	FetchAllChainIDsByName(chainName [][]byte) (chainIDs *[]notaryapi.Hash, err error)	{
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	name := notaryapi.EncodeChainNameToString(chainName)
	
	var fromkey [] byte = []byte{byte(TBL_CHAIN_NAME)} 		  		// Table Name (1 bytes)
	fromkey = append(fromkey, []byte(name) ...) 					// Chain Type (32 bytes)
	
	chainIDSlice := make([]notaryapi.Hash, 0, 10) 	
	
	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: nil}, db.ro)
	
	for iter.Next() {		
		chainID := new (notaryapi.Hash)
		chainID.Bytes = make([]byte, len(iter.Value()))		
		copy(chainID.Bytes, iter.Value())
		chainIDSlice = append(chainIDSlice, *chainID)		
	}
	iter.Release()
	err = iter.Error()
	
	return &chainIDSlice, nil	
}