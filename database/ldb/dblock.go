package ldb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"log"
)

// FetchDBEntriesFromQueue gets all of the dbentries that have not been processed
/*func (db *LevelDb) FetchDBEntriesFromQueue(startTime *[]byte) (dbentries []*common.DBEntry, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_EB_QUEUE)} // Table Name (1 bytes)
	fromkey = append(fromkey, *startTime...)        // Timestamp  (8 bytes)

	var tokey []byte = []byte{byte(TBL_EB_QUEUE)} // Table Name (4 bytes)
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(time.Now().Unix()))
	tokey = append(tokey, binaryTimestamp...) // Timestamp  (8 bytes)

	fbEntrySlice := make([]*common.DBEntry, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		if bytes.Equal(iter.Value(), []byte{byte(STATUS_IN_QUEUE)}) {
			key := make([]byte, len(iter.Key()))
			copy(key, iter.Key())
			dbEntry := new(common.DBEntry)

			dbEntry.SetTimeStamp(key[1:9]) // Timestamp (8 bytes)
			cid := key[9:41]
			dbEntry.ChainID = new(common.Hash)
			dbEntry.ChainID.Bytes = cid // Chain id (32 bytes)
			dbEntry.SetHash(key[41:73]) // Entry Hash (32 bytes)

			fbEntrySlice = append(fbEntrySlice, dbEntry)
		}
	}
	iter.Release()
	err = iter.Error()

	return fbEntrySlice, nil
}
*/
// ProcessDBlockBatche inserts the DBlock and update all it's dbentries in DB
func (db *LevelDb) ProcessDBlockBatch(dblock *common.DirectoryBlock) error {

	if dblock != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		binaryDblock, err := dblock.MarshalBinary()
		if err != nil {
			return err
		}

		if dblock.DBHash == nil {
			dblock.DBHash = common.Sha(binaryDblock)
		}

		// Insert the binary directory block
		var key []byte = []byte{byte(TBL_DB)}
		key = append(key, dblock.DBHash.Bytes...)
		db.lbatch.Put(key, binaryDblock)

		// Insert block height cross reference
		var dbNumkey []byte = []byte{byte(TBL_DB_NUM)}
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, dblock.Header.BlockHeight)
		dbNumkey = append(dbNumkey, buf.Bytes()...)
		db.lbatch.Put(dbNumkey, dblock.DBHash.Bytes)

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// Insert the Directory Block meta data into db
func (db *LevelDb) InsertDBInfo(dbInfo common.DBInfo) (err error) {
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
func (db *LevelDb) FetchDBInfoByHash(dbHash *common.Hash) (dbInfo *common.DBInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB_INFO)}
	key = append(key, dbHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		dbInfo = new(common.DBInfo)
		dbInfo.UnmarshalBinary(data)
	}

	return dbInfo, nil
}

// FetchDBlock gets an entry by hash from the database.
func (db *LevelDb) FetchDBlockByHash(dBlockHash *common.Hash) (dBlock *common.DirectoryBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB)}
	key = append(key, dBlockHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data == nil {
		return nil, errors.New("DBlock not found for Hash: " + dBlockHash.String())
	} else {
		dBlock = new(common.DirectoryBlock)
		dBlock.UnmarshalBinary(data)
	}

	log.Println("dBlock.Header.MerkleRoot: ", dBlock.Header.BodyMR.String())

	for _, entry := range dBlock.DBEntries {
		log.Println("entry.MerkleRoot: ", entry.MerkleRoot.String())
	}

	return dBlock, nil
}

// FetchDBlockByHeight gets an directory block by height from the database.
func (db *LevelDb) FetchDBlockByHeight(dBlockHeight uint32) (dBlock *common.DirectoryBlock, err error) {
	dBlockHash, _ := db.FetchDBHashByHeight(dBlockHeight)

	if dBlockHash != nil {
		dBlock, _ = db.FetchDBlockByHash(dBlockHash)
	}

	return dBlock, nil
}

// FetchDBHashByHeight gets a dBlockHash from the database.
func (db *LevelDb) FetchDBHashByHeight(dBlockHeight uint32) (dBlockHash *common.Hash, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB_NUM)}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, dBlockHeight)
	key = append(key, buf.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		dBlockHash = new(common.Hash)
		dBlockHash.UnmarshalBinary(data)
	}
	return dBlockHash, nil
}

// FetchAllDBInfo gets all of the fbInfo
func (db *LevelDb) FetchAllDBlocks() (dBlocks []common.DirectoryBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_DB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_DB + 1)} // Table Name (1 bytes)

	dBlockSlice := make([]common.DirectoryBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		var dBlock common.DirectoryBlock
		dBlock.UnmarshalBinary(iter.Value())
		dBlock.DBHash = common.Sha(iter.Value()) //to be optimized??

		dBlockSlice = append(dBlockSlice, dBlock)

	}
	iter.Release()
	err = iter.Error()

	return dBlockSlice, nil
}
