package ldb

import (
	"errors"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/old/oldcoin"
	//	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	//	"log"
)

// ProcessFBlockBatche inserts the FBlock
func (db *LevelDb) ProcessFBlockBatch(block *oldcoin.FBlock) error {

	/*
		if block != nil {
			if db.lbatch == nil {
				db.lbatch = new(leveldb.Batch)
			}

			defer db.lbatch.Reset()

			if len(block.Transactions) < 1 {
				return errors.New("Empty dblock!")
			}

			binaryBlock, err := block.MarshalBinary()
			if err != nil {
				return err
			}

			// Insert the binary factom block
			var key []byte = []byte{byte(TBL_FB)}
			key = append(key, block.FBHash.Bytes...)
			db.lbatch.Put(key, binaryBlock)

			err = db.lDb.Write(db.lbatch, db.wo)
			if err != nil {
				log.Println("batch failed %v\n", err)
				return err
			}

		}
		return nil
	*/
	return errors.New("NOT IMPLEMENTED !!!!!!!!!!!!!!!!!!!!")
}

// FetchCntryBlock gets a block by hash from the database.
func (db *LevelDb) FetchFBlockByHash(fBlockHash *notaryapi.Hash) (fBlock *oldcoin.FBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_FB)}
	key = append(key, fBlockHash.Bytes...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		fBlock = new(oldcoin.FBlock)
		//		fBlock.UnmarshalBinary(data)
	}
	return fBlock, nil
}

// FetchAllFBlocks gets all of the oldcoin blocks
func (db *LevelDb) FetchAllFBlocks() (fBlocks []oldcoin.FBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_FB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_FB + 1)} // Table Name (1 bytes)

	fBlockSlice := make([]oldcoin.FBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		var fBlock oldcoin.FBlock
		//		fBlock.UnmarshalBinary(iter.Value())
		fBlock.FBHash = notaryapi.Sha(iter.Value()) //to be optimized??

		fBlockSlice = append(fBlockSlice, fBlock)

	}
	iter.Release()
	err = iter.Error()

	return fBlockSlice, nil
}
