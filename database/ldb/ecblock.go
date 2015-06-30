package ldb

import (
//	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"log"
)

// ProcessECBlockBatche inserts the ECBlock and update all it's cbentries in DB
func (db *LevelDb) ProcessECBlockBatch(block *common.ECBlock) error {

	if block != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		binaryBlock, err := block.MarshalBinary()
		if err != nil {
			return err
		}

		// Insert the binary factom block
		var key []byte = []byte{byte(TBL_CB)}
		key = append(key, block.KeyMR().Bytes()...)
		db.lbatch.Put(key, binaryBlock)

		// Update the chain head reference
		key = []byte{byte(TBL_CHAIN_HEAD)}
		key = append(key, common.EC_CHAINID...)
		db.lbatch.Put(key, block.KeyMR().Bytes())	
		
		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchECBlockByHash gets an Entry Credit block by hash from the database.
func (db *LevelDb) FetchECBlockByHash(ecBlockHash *common.Hash) (ecBlock *common.ECBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_CB)}
	key = append(key, ecBlockHash.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		ecBlock = common.NewECBlock()
		ecBlock.UnmarshalBinary(data)
	}
	return ecBlock, nil
}

// FetchAllECBlocks gets all of the entry credit blocks
func (db *LevelDb) FetchAllECBlocks() (ecBlocks []common.ECBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_CB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_CB + 1)} // Table Name (1 bytes)

	ecBlockSlice := make([]common.ECBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		ecBlock := common.NewECBlock()
		ecBlock.UnmarshalBinary(iter.Value())
		ecBlockSlice = append(ecBlockSlice, *ecBlock)
	}
	iter.Release()
	err = iter.Error()

	return ecBlockSlice, nil
}
