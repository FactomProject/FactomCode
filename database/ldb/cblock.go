package ldb

import (
	"errors"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"log"
)

// ProcessCBlockBatche inserts the CBlock and update all it's cbentries in DB
func (db *LevelDb) ProcessCBlockBatch(block *notaryapi.CBlock) error {

	if block != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		if len(block.CBEntries) < 1 {
			return errors.New("Empty dblock!")
		}

		binaryBlock, err := block.MarshalBinary()
		if err != nil {
			return err
		}

		// Insert the binary factom block
		var key []byte = []byte{byte(TBL_CB)}
		key = append(key, block.CBHash.Bytes...)
		db.lbatch.Put(key, binaryBlock)

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchCntryBlock gets a block by hash from the database.
func (db *LevelDb) FetchCBlockByHash(cBlockHash *notaryapi.Hash) (cBlock *notaryapi.CBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_CB)}
	key = append(key, cBlockHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		cBlock = new(notaryapi.CBlock)
		cBlock.UnmarshalBinary(data)
	}
	return cBlock, nil
}

// FetchAllCBlocks gets all of the entry credit blocks
func (db *LevelDb) FetchAllCBlocks() (cBlocks []notaryapi.CBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_CB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_CB + 1)} // Table Name (1 bytes)

	cBlockSlice := make([]notaryapi.CBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		var cBlock notaryapi.CBlock
		cBlock.UnmarshalBinary(iter.Value())
		cBlock.CBHash = notaryapi.Sha(iter.Value()) //to be optimized??

		cBlockSlice = append(cBlockSlice, cBlock)

	}
	iter.Release()
	err = iter.Error()

	return cBlockSlice, nil
}
