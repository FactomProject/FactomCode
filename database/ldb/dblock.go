package ldb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"log"
	"time"
)

// FetchDBEntriesFromQueue gets all of the dbentries that have not been processed
func (db *LevelDb) FetchDBEntriesFromQueue(startTime *[]byte) (dbentries []*notaryapi.DBEntry, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_EB_QUEUE)} // Table Name (1 bytes)
	fromkey = append(fromkey, *startTime...)        // Timestamp  (8 bytes)

	var tokey []byte = []byte{byte(TBL_EB_QUEUE)} // Table Name (4 bytes)
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(time.Now().Unix()))
	tokey = append(tokey, binaryTimestamp...) // Timestamp  (8 bytes)

	fbEntrySlice := make([]*notaryapi.DBEntry, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		if bytes.Equal(iter.Value(), []byte{byte(STATUS_IN_QUEUE)}) {
			key := make([]byte, len(iter.Key()))
			copy(key, iter.Key())
			dbEntry := new(notaryapi.DBEntry)

			dbEntry.SetTimeStamp(key[1:9]) // Timestamp (8 bytes)
			cid := key[9:41]
			dbEntry.ChainID = new(notaryapi.Hash)
			dbEntry.ChainID.Bytes = cid // Chain id (32 bytes)
			dbEntry.SetHash(key[41:73]) // Entry Hash (32 bytes)

			fbEntrySlice = append(fbEntrySlice, dbEntry)
		}
	}
	iter.Release()
	err = iter.Error()

	return fbEntrySlice, nil
}

// ProcessDBlockBatche inserts the DBlock and update all it's dbentries in DB
func (db *LevelDb) ProcessDBlockBatch(dblock *notaryapi.DBlock) error {

	if dblock != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		if len(dblock.DBEntries) < 1 {
			return errors.New("Empty dblock!")
		}

		binaryDblock, err := dblock.MarshalBinary()
		if err != nil {
			return err
		}

		// Insert the binary directory block
		var key []byte = []byte{byte(TBL_DB)}
		key = append(key, dblock.DBHash.Bytes...)
		db.lbatch.Put(key, binaryDblock)

		// Insert block height cross reference
		var dbNumkey []byte = []byte{byte(TBL_DB_NUM)}
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, dblock.Header.BlockID)
		dbNumkey = append(dbNumkey, buf.Bytes()...)
		db.lbatch.Put(dbNumkey, dblock.DBHash.Bytes)

		// Update DBEntry process queue for each dbEntry in dblock
		for i := 0; i < len(dblock.DBEntries); i++ {
			var dbEntry notaryapi.DBEntry = *dblock.DBEntries[i]
			var fbEntryKey []byte = []byte{byte(TBL_EB_QUEUE)}               // Table Name (1 bytes)
			fbEntryKey = append(fbEntryKey, dbEntry.GetBinaryTimeStamp()...) // Timestamp (8 bytes)
			fbEntryKey = append(fbEntryKey, dbEntry.ChainID.Bytes...)        // Chain id (32 bytes)
			fbEntryKey = append(fbEntryKey, dbEntry.Hash().Bytes...)         // Entry Hash (32 bytes)
			db.lbatch.Put(fbEntryKey, []byte{byte(STATUS_PROCESSED)})

			if isLookupDB {
				// Create an EBInfo and insert it into db
				var ebInfo = new(notaryapi.EBInfo)
				ebInfo.EBHash = dbEntry.Hash()
				ebInfo.MerkleRoot = dbEntry.MerkleRoot
				ebInfo.DBHash = dblock.DBHash
				ebInfo.DBBlockNum = dblock.Header.BlockID
				ebInfo.ChainID = dbEntry.ChainID
				var ebInfoKey []byte = []byte{byte(TBL_EB_INFO)}
				ebInfoKey = append(ebInfoKey, ebInfo.EBHash.Bytes...)
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

// Insert the Directory Block meta data into db
func (db *LevelDb) InsertDBBatch(dbBatch *notaryapi.DBBatch) (err error) {
	if dbBatch.BTCBlockHash == nil || dbBatch.BTCTxHash == nil {
		return
	}

	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	// Insert DBBatch - using the first block id in the batch as the key
	var Key []byte = []byte{byte(TBL_DBATCH)}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, dbBatch.DBlocks[0].Header.BlockID)
	Key = append(Key, buf.Bytes()...)

	binaryDBBatch, _ := dbBatch.MarshalBinary()
	db.lbatch.Put(Key, binaryDBBatch)

	// Insert  dblock - DBBatch cross reference
	for _, dblock := range dbBatch.DBlocks {
		var dbKey []byte = []byte{byte(TBL_DB_BATCH)} // Table Name (1 bytes)
		dbKey = append(dbKey, dblock.DBHash.Bytes...)
		db.lbatch.Put(dbKey, buf.Bytes())
	}

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}

	return nil
}

// FetchDBBatchByHash gets an DBBatch obj
func (db *LevelDb) FetchDBBatchByHash(dbHash *notaryapi.Hash) (dbBatch *notaryapi.DBBatch, err error) {

	dBlockID, _ := db.FetchDBlockIDByHash(dbHash)
	if dBlockID > -1 {
		dbBatch, err = db.FetchDBBatchByDBlockID(uint64(dBlockID))
	}

	return dbBatch, err
}

// FetchDBBatchByDBlockID gets an DBBatch obj
func (db *LevelDb) FetchDBBatchByDBlockID(dBlockID uint64) (dbBatch *notaryapi.DBBatch, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, dBlockID)

	var key []byte = []byte{byte(TBL_DBATCH)}
	key = append(key, buf.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		dbBatch = new(notaryapi.DBBatch)
		dbBatch.UnmarshalBinary(data)
	}

	return dbBatch, nil
}

// FetchDBlockIDByHash gets an dBlockID
func (db *LevelDb) FetchDBlockIDByHash(dbHash *notaryapi.Hash) (dBlockID int64, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB_BATCH)}
	key = append(key, dbHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		dBlockID = int64(binary.BigEndian.Uint64(data[:8]))
	} else {
		dBlockID = int64(-1)
	}

	return dBlockID, nil
}

// FetchDBlock gets an entry by hash from the database.
func (db *LevelDb) FetchDBlockByHash(dBlockHash *notaryapi.Hash) (dBlock *notaryapi.DBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB)}
	key = append(key, dBlockHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data == nil {
		return nil, errors.New("DBlock not found for Hash: " + dBlockHash.String())
	} else {
		dBlock = new(notaryapi.DBlock)
		dBlock.UnmarshalBinary(data)
	}

	log.Println("dBlock.Header.MerkleRoot:%v", dBlock.Header.MerkleRoot.String())

	for _, entry := range dBlock.DBEntries {
		log.Println("entry.MerkleRoot:%v", entry.MerkleRoot.String())
	}

	return dBlock, nil
}

// FetchAllDBInfo gets all of the fbInfo
func (db *LevelDb) FetchAllDBlocks() (dBlocks []notaryapi.DBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_DB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_DB + 1)} // Table Name (1 bytes)

	dBlockSlice := make([]notaryapi.DBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		var dBlock notaryapi.DBlock
		dBlock.UnmarshalBinary(iter.Value())
		dBlock.DBHash = notaryapi.Sha(iter.Value()) //to be optimized??

		dBlockSlice = append(dBlockSlice, dBlock)

	}
	iter.Release()
	err = iter.Error()

	return dBlockSlice, nil
}

// FetchDBInfoByHash gets an DBInfo obj
func (db *LevelDb) FetchAllDBRecordsByDBHash(dbHash *notaryapi.Hash, cChainID *notaryapi.Hash) (ldbMap map[string]string, err error) {
	//	db.dbLock.Lock()
	//	defer db.dbLock.Unlock()
	if ldbMap == nil {
		ldbMap = make(map[string]string)
	}

	var dblock notaryapi.DBlock
	var eblock notaryapi.EBlock

	//DBlock
	var key []byte = []byte{byte(TBL_DB)}
	key = append(key, dbHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data == nil {
		return nil, errors.New("DBlock not found for DBHash: " + dbHash.String())
	} else {
		dblock.UnmarshalBinary(data)
		ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
	}

	f, _ := db.FetchEBInfoByHash(dbHash)
	if f == nil {
		log.Println("f is null")
	}

	// Chain Num cross references --- to be removed ???
	fromkey := []byte{byte(TBL_EB_CHAIN_NUM)}
	tokey := []byte{byte(TBL_EB_CHAIN_NUM + 1)}
	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		ldbMap[notaryapi.EncodeBinary(&k)] = notaryapi.EncodeBinary(&v)
	}
	iter.Release()
	err = iter.Error()

	//EBlocks or CBlock or FBlock
	for _, dbEntry := range dblock.DBEntries {

		//CBlock
		if dbEntry.ChainID.IsSameAs(cChainID) {
			key = []byte{byte(TBL_CB)}
			key = append(key, dbEntry.MerkleRoot.Bytes...)
			data, err = db.lDb.Get(key, db.ro)

			if data == nil {
				return nil, errors.New("CBlock not found for cBHash: " + dbEntry.MerkleRoot.String())
			} else {
				ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
			}
			continue
		}

		//EB Merkle root and EBHash Cross Reference
		key = []byte{byte(TBL_EB_MR)}
		key = append(key, dbEntry.MerkleRoot.Bytes...)
		data, err = db.lDb.Get(key, db.ro)
		if data == nil {
			return nil, errors.New("EBHash not found for MR: " + dbEntry.MerkleRoot.String())
		} else {
			dbHash := new(notaryapi.Hash)
			dbHash.UnmarshalBinary(data)
			dbEntry.SetHash(dbHash.Bytes)
			ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
		}

		//EBlock
		key = []byte{byte(TBL_EB)}
		key = append(key, dbEntry.Hash().Bytes...)
		data, err = db.lDb.Get(key, db.ro)
		if data == nil {
			return nil, errors.New("EBlock not found for EBHash: " + dbEntry.Hash().String())
		} else {
			eblock.UnmarshalBinary(data)
			ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
		}
		//EBInfo
		key = []byte{byte(TBL_EB_INFO)}
		key = append(key, dbEntry.Hash().Bytes...)
		data, err = db.lDb.Get(key, db.ro)
		if data == nil {
			return nil, errors.New("EBInfo not found for EBHash: " + dbEntry.Hash().String())
		} else {
			ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
		}

		//Entries
		for _, ebentry := range eblock.EBEntries {
			//Entry
			key = []byte{byte(TBL_ENTRY)}
			key = append(key, ebentry.Hash().Bytes...)
			data, err = db.lDb.Get(key, db.ro)
			if data == nil {
				return nil, errors.New("Entry not found for entry hash: " + ebentry.Hash().String())
			} else {
				ldbMap[notaryapi.EncodeBinary(&key)] = notaryapi.EncodeBinary(&data)
			}
			//EntryInfo
			key = []byte{byte(TBL_ENTRY_INFO)}
			key = append(key, ebentry.Hash().Bytes...)
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

// FetchSupportDBRecords gets all supporting db records
func (db *LevelDb) FetchSupportDBRecords() (ldbMap map[string]string, err error) {
	//	db.dbLock.Lock()
	//	defer db.dbLock.Unlock()
	if ldbMap == nil {
		ldbMap = make(map[string]string)
	}

	// Chains
	fromkey := []byte{byte(TBL_CHAIN_HASH)}
	tokey := []byte{byte(TBL_CHAIN_HASH + 1)}
	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		ldbMap[notaryapi.EncodeBinary(&k)] = notaryapi.EncodeBinary(&v)
	}
	iter.Release()
	err = iter.Error()

	// Chain Name cross references
	fromkey = []byte{byte(TBL_CHAIN_NAME)}
	tokey = []byte{byte(TBL_CHAIN_NAME + 1)}
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		ldbMap[notaryapi.EncodeBinary(&k)] = notaryapi.EncodeBinary(&v)
	}
	iter.Release()
	err = iter.Error()

	// Chain Num cross references
	fromkey = []byte{byte(TBL_EB_CHAIN_NUM)}
	tokey = []byte{byte(TBL_EB_CHAIN_NUM + 1)}
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		ldbMap[notaryapi.EncodeBinary(&k)] = notaryapi.EncodeBinary(&v)
	}
	iter.Release()
	err = iter.Error()

	//DBBatch -- to be optimized??
	fromkey = []byte{byte(TBL_DBATCH)}
	tokey = []byte{byte(TBL_DBATCH + 1)}
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		ldbMap[notaryapi.EncodeBinary(&k)] = notaryapi.EncodeBinary(&v)
	}
	iter.Release()
	err = iter.Error()

	// DB_Batch -- to be optimized??
	fromkey = []byte{byte(TBL_DB_BATCH)}
	tokey = []byte{byte(TBL_DB_BATCH + 1)}
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		ldbMap[notaryapi.EncodeBinary(&k)] = notaryapi.EncodeBinary(&v)
	}
	iter.Release()
	err = iter.Error()

	return ldbMap, nil
}

// InsertAllDBRecords inserts all key value pairs from map into db
func (db *LevelDb) InsertAllDBRecords(ldbMap map[string]string) (err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	for key, value := range ldbMap {
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
