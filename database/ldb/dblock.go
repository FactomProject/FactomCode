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
func (db *LevelDb) InsertDBInfo(dbInfo notaryapi.DBInfo) (err error) {
	if dbInfo.BTCBlockHash == nil || dbInfo.BTCTxHash == nil {
		return
	}

	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	var key []byte = []byte{byte(TBL_DB_INFO)} // Table Name (1 bytes)
	key = append(key, dbInfo.DBHash.Bytes...)
	binaryDBInfo, _ := dbInfo.MarshalBinary()
	db.lbatch.Put(key, binaryDBInfo)

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}

	return nil
}

// FetchDBInfoByHash gets an DBInfo obj
func (db *LevelDb) FetchDBInfoByHash(dbHash *notaryapi.Hash) (dbInfo *notaryapi.DBInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB_INFO)}
	key = append(key, dbHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		dbInfo = new(notaryapi.DBInfo)
		dbInfo.UnmarshalBinary(data)
	}

	return dbInfo, nil
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
		dBlock.DBHash = dBlockHash
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
