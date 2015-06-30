package ldb

import (
//	"errors"
    "github.com/FactomProject/factoid/block"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"encoding/binary"	
	"log"
)

// ProcessFBlockBatch inserts the factoid block
func (db *LevelDb) ProcessFBlockBatch(block block.IFBlock) error {

	if block != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		binaryBlock, err := block.MarshalBinary()
		if err != nil {
			return err
		}

        scHash := common.Sha(binaryBlock)
		
		// Insert the binary factom block
		var key []byte = []byte{byte(TBL_SC)}
		key = append(key, scHash.Bytes()...)
		db.lbatch.Put(key, binaryBlock)
		
		// Insert the sc block number cross reference
		key = []byte{byte(TBL_SC_NUM)}
		key = append(key, block.GetChainID().Bytes()...)
		bytes := make([]byte, 4)
        binary.BigEndian.PutUint32(bytes, block.GetDBHeight())
		key = append(key, bytes...)
        db.lbatch.Put(key, scHash.Bytes())		
        
		// Update the chain head reference
		key = []byte{byte(TBL_CHAIN_HEAD)}
		key = append(key, common.FACTOID_CHAINID...)
		db.lbatch.Put(key,scHash.Bytes())	        

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchFBlockByHash gets an factoid block by hash from the database.
func (db *LevelDb) FetchFBlockByHash(hash *common.Hash) ( FBlock block.IFBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_SC)}
	key = append(key, hash.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		FBlock = new(block.FBlock)
		FBlock.UnmarshalBinary(data)
	}
	return FBlock, nil
}

// FetchAllFBlocks gets all of the factoid blocks
func (db *LevelDb) FetchAllFBlocks() (FBlocks []block.IFBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_SC)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey []byte = []byte{byte(TBL_SC + 1)} // Table Name (1 bytes)

	FBlockSlice := make([]block.IFBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		FBlock := new(block.FBlock)
		FBlock.UnmarshalBinary(iter.Value())
		
		FBlockSlice = append(FBlockSlice, FBlock)

	} 
	iter.Release()
	err = iter.Error()

	return FBlockSlice, nil
}
