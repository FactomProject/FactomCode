package ldb

import (
	//	"errors"
	"bytes"
	"encoding/binary"
	"log"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/iterator"
	"github.com/FactomProject/goleveldb/leveldb/util"
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

		abHash, err := block.PartialHash()
		if err != nil {
			return err
		}

		// Insert the binary factom block
		var key = []byte{byte(TBL_AB)}
		key = append(key, abHash.Bytes()...)
		db.lbatch.Put(key, binaryBlock)

		// Insert the admin block number cross reference
		key = []byte{byte(TBL_AB_NUM)}
		key = append(key, common.ADMIN_CHAINID...)
		bytes := make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, block.Header.DBHeight)
		key = append(key, bytes...)
		db.lbatch.Put(key, abHash.Bytes())

		// Update the chain head reference
		key = []byte{byte(TBL_CHAIN_HEAD)}
		key = append(key, common.ADMIN_CHAINID...)
		db.lbatch.Put(key, abHash.Bytes())

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Printf("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchABlockByHash gets an admin block by hash from the database.
func (db *LevelDb) FetchABlockByHash(aBlockHash *common.Hash) (aBlock *common.AdminBlock, err error) {
	var key = []byte{byte(TBL_AB)}
	key = append(key, aBlockHash.Bytes()...)
	var data []byte
	db.dbLock.Lock()
	data, err = db.lDb.Get(key, db.ro)
	db.dbLock.Unlock()

	if data != nil {
		aBlock = new(common.AdminBlock)
		_, err := aBlock.UnmarshalBinaryData(data)
		if err != nil {
			return nil, err
		}
	}
	return aBlock, nil
}

// FetchABlockByHeight gets an admin block by hash from the database.
func (db *LevelDb) FetchABlockByHeight(height uint32) (aBlock *common.AdminBlock, err error) {
	var key = []byte{byte(TBL_AB_NUM)}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, height)
	key = append(key, common.ADMIN_CHAINID...)
	key = append(key, buf.Bytes()...)

	var data []byte
	db.dbLock.Lock()
	data, err = db.lDb.Get(key, db.ro)
	db.dbLock.Unlock()
	if err != nil {
		return nil, err
	}

	aBlockHash := common.NewHash()
	_, err = aBlockHash.UnmarshalBinaryData(data)
	if err != nil {
		return nil, err
	}
	return db.FetchABlockByHash(aBlockHash)
}

// FetchAllABlocks gets all of the admin blocks
func (db *LevelDb) FetchAllABlocks() (aBlocks []common.AdminBlock, err error) {
	var fromkey = []byte{byte(TBL_AB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey = []byte{byte(TBL_AB + 1)} // Table Name (1 bytes)
	var iter iterator.Iterator
	aBlockSlice := make([]common.AdminBlock, 0, 10)
	db.dbLock.Lock()
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	db.dbLock.Unlock()

	for iter.Next() {
		var aBlock common.AdminBlock
		_, err := aBlock.UnmarshalBinaryData(iter.Value())
		if err != nil {
			return nil, err
		}
		//TODO: to be optimized??
		_, err = aBlock.PartialHash()
		if err != nil {
			return nil, err
		}

		aBlockSlice = append(aBlockSlice, aBlock)

	}
	iter.Release()
	err = iter.Error()

	return aBlockSlice, nil
}
