package ldb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"github.com/FactomProject/btcd/wire"
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

		if dblock.KeyMR == nil {
			dblock.BuildKeyMerkleRoot()
		}

		// Insert the binary directory block
		var key []byte = []byte{byte(TBL_DB)}
		key = append(key, dblock.DBHash.Bytes()...)
		db.lbatch.Put(key, binaryDblock)

		// Insert block height cross reference
		var dbNumkey []byte = []byte{byte(TBL_DB_NUM)}
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, dblock.Header.BlockHeight)
		dbNumkey = append(dbNumkey, buf.Bytes()...)
		db.lbatch.Put(dbNumkey, dblock.DBHash.Bytes())

		// Insert the directory block merkle root cross reference
		key = []byte{byte(TBL_DB_MR)}
		key = append(key, dblock.KeyMR.Bytes()...)
		binaryDBHash, _ := dblock.DBHash.MarshalBinary()
		db.lbatch.Put(key, binaryDBHash)

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

		// Update DirBlock Height cache
		db.lastDirBlkHeight = int64(dblock.Header.BlockHeight)
		db.lastDirBlkSha, _ = wire.NewShaHash(dblock.DBHash.Bytes())
		db.lastDirBlkShaCached = true

	}
	return nil
}

// UpdateBlockHeightCache updates the dir block height cache in db
func (db *LevelDb) UpdateBlockHeightCache(dirBlkHeigh uint32, dirBlkHash *common.Hash) error {

	// Update DirBlock Height cache
	db.lastDirBlkHeight = int64(dirBlkHeigh)
	db.lastDirBlkSha, _ = wire.NewShaHash(dirBlkHash.Bytes())
	db.lastDirBlkShaCached = true
	return nil
}

// FetchBlockHeightCache returns the hash and block height of the most recent
func (db *LevelDb)	FetchBlockHeightCache() (sha *wire.ShaHash, height int64, err error) {
	return db.lastDirBlkSha, db.lastDirBlkHeight, nil
}


// Insert the Directory Block meta data into db
func (db *LevelDb) InsertDirBlockInfo(dirBlockInfo *common.DirBlockInfo) (err error) {
	if dirBlockInfo.BTCTxHash == nil {
		return
	}

	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	var key []byte = []byte{byte(TBL_DB_INFO)} // Table Name (1 bytes)
	key = append(key, dirBlockInfo.DBHash.Bytes()...)
	binaryDirBlockInfo, _ := dirBlockInfo.MarshalBinary()
	db.lbatch.Put(key, binaryDirBlockInfo)

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}

	return nil
}

// FetchDirBlockInfoByHash gets an DirBlockInfo obj
func (db *LevelDb) FetchDirBlockInfoByHash(dbHash *common.Hash) (dirBlockInfo *common.DirBlockInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB_INFO)}
	key = append(key, dbHash.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		dirBlockInfo = new(common.DirBlockInfo)
		dirBlockInfo.UnmarshalBinary(data)
	}

	return dirBlockInfo, nil
}

// FetchDBlock gets an entry by hash from the database.
func (db *LevelDb) FetchDBlockByHash(dBlockHash *common.Hash) (dBlock *common.DirectoryBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB)}
	key = append(key, dBlockHash.Bytes()...)
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

// FetchDBHashByMR gets a DBHash by MR from the database.
func (db *LevelDb) FetchDBHashByMR(dBMR *common.Hash) (*common.Hash, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_DB_MR)}
	key = append(key, dBMR.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)
	if err != nil {
		return nil, err
	}

	dBlockHash := common.NewHash()
	if err := dBlockHash.UnmarshalBinary(data); err != nil {
		return dBlockHash, err
	}

	return dBlockHash, nil
}

// FetchDBlockByMR gets a directory block by merkle root from the database.
func (db *LevelDb) FetchDBlockByMR(dBMR *common.Hash) (*common.DirectoryBlock, error) {
	dBlockHash, err := db.FetchDBHashByMR(dBMR)
	if err != nil {
		return nil, err
	}

	dBlock, err := db.FetchDBlockByHash(dBlockHash)
	if err != nil {
		return dBlock, err
	}

	return dBlock, nil
}

// FetchAllDBlocks gets all of the fbInfo
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

// FetchAllDirBlockInfo gets all of the dirBlockInfo
func (db *LevelDb) FetchAllDirBlockInfo() (dirBlockInfoMap map[string]*common.DirBlockInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_DB_INFO)}   // Table Name (1 bytes)
	var tokey []byte = []byte{byte(TBL_DB_INFO + 1)} // Table Name (1 bytes)

	dirBlockInfoMap = make(map[string]*common.DirBlockInfo)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		dBInfo := new(common.DirBlockInfo)
		dBInfo.UnmarshalBinary(iter.Value())
		dirBlockInfoMap[dBInfo.DBMerkleRoot.String()] = dBInfo
	}
	iter.Release()
	err = iter.Error()
	return dirBlockInfoMap, err
}

// FetchAllUnconfirmedDirBlockInfo gets all of the dirBlockInfos that have BTC Anchor confirmation
func (db *LevelDb) FetchAllUnconfirmedDirBlockInfo() (dirBlockInfoMap map[string]*common.DirBlockInfo, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_DB_INFO)}   // Table Name (1 bytes)
	var tokey []byte = []byte{byte(TBL_DB_INFO + 1)} // Table Name (1 bytes)

	dirBlockInfoMap = make(map[string]*common.DirBlockInfo)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		dBInfo := new(common.DirBlockInfo)

		// The last byte stores the confirmation flag
		if iter.Value()[len(iter.Value())-1] == 0 {
			dBInfo.UnmarshalBinary(iter.Value())
			dirBlockInfoMap[dBInfo.DBMerkleRoot.String()] = dBInfo
		}
	}
	iter.Release()
	err = iter.Error()
	return dirBlockInfoMap, err
}
