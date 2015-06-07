package ldb

import (
//	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"encoding/binary"	
	"log"
)

// ProcessABlockBatch inserts the AdminBlock
func (db *LevelDb) ProcessABlockBatch(block *common.AdminBlock) error {

	if block != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		binaryBlock, err := block.MarshalBinary()
		if err != nil {
			return err
		}

		if block.ABHash == nil {
			block.ABHash = common.Sha(binaryBlock)
		}
		
		// Insert the binary factom block
		var key []byte = []byte{byte(TBL_AB)}
		key = append(key, block.ABHash.Bytes()...)
		db.lbatch.Put(key, binaryBlock)
		
		// Insert the admin block number cross reference
		key = []byte{byte(TBL_AB_NUM)}
		key = append(key, block.Header.ChainID.Bytes()...)
		bytes := make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, block.Header.DBHeight)
		key = append(key, bytes...)
		db.lbatch.Put(key, block.ABHash.Bytes())		

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchABlockByHash gets an admin block by hash from the database.
func (db *LevelDb) FetchABlockByHash(aBlockHash *common.Hash) (aBlock *common.AdminBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_AB)}
	key = append(key, aBlockHash.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		aBlock = new(common.AdminBlock)
		aBlock.UnmarshalBinary(data)
	}
	return aBlock, nil
}

// FetchAllABlocks gets all of the admin blocks
func (db *LevelDb) FetchAllABlocks() (aBlocks []common.AdminBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_AB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_AB + 1)} // Table Name (1 bytes)

	aBlockSlice := make([]common.AdminBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		var aBlock common.AdminBlock
		aBlock.UnmarshalBinary(iter.Value())
		aBlock.ABHash = common.Sha(iter.Value()) //to be optimized??

		aBlockSlice = append(aBlockSlice, aBlock)

	} 
	iter.Release()
	err = iter.Error()

	return aBlockSlice, nil
}
