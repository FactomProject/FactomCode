package ldb

import (
	//	"errors"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/iterator"
	"github.com/FactomProject/goleveldb/leveldb/util"
	"github.com/davecgh/go-spew/spew"
)

// ProcessECBlockBatch inserts the ECBlock and update all it's cbentries in DB
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
		var key = []byte{byte(TBL_CB)}
		hash, err := block.HeaderHash()
		if err != nil {
			return err
		}
		key = append(key, hash.Bytes()...)
		db.lbatch.Put(key, binaryBlock)

		// Insert block height cross reference
		var dbNumkey = []byte{byte(TBL_CB_NUM)}
		dbNumkey = append(dbNumkey, common.EC_CHAINID...)
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, block.Header.EBHeight)
		dbNumkey = append(dbNumkey, buf.Bytes()...)
		db.lbatch.Put(dbNumkey, hash.Bytes())
		fmt.Println("ProcessECBlockBatch: key=", hex.EncodeToString(dbNumkey), ", hash=", hash)

		// Update the chain head reference
		key = []byte{byte(TBL_CHAIN_HEAD)}
		key = append(key, common.EC_CHAINID...)
		//hash, err = block.HeaderHash()
		//if err != nil {
		//return err
		//}
		db.lbatch.Put(key, hash.Bytes())

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Printf("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchECBlockByHash gets an Entry Credit block by hash from the database.
func (db *LevelDb) FetchECBlockByHash(ecBlockHash *common.Hash) (ecBlock *common.ECBlock, err error) {
	var key = []byte{byte(TBL_CB)}
	key = append(key, ecBlockHash.Bytes()...)
	var data []byte
	db.dbLock.Lock()
	data, err = db.lDb.Get(key, db.ro)
	db.dbLock.Unlock()
	if err != nil {
		return nil, err
	}
	fmt.Println("FetchECBlockByHash: key=", hex.EncodeToString(key), ", data=", string(data))

	if data != nil {
		ecBlock = common.NewECBlock()
		_, err := ecBlock.UnmarshalBinaryData(data)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("FetchECBlockByHash: ecBlock=", spew.Sdump(ecBlock))
	return ecBlock, nil
}

// FetchECBlockByHeight gets an Entry Credit block by hash from the database.
func (db *LevelDb) FetchECBlockByHeight(height uint32) (ecBlock *common.ECBlock, err error) {
	var key = []byte{byte(TBL_CB_NUM)}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, height)
	key = append(key, common.EC_CHAINID...)
	key = append(key, buf.Bytes()...)
	fmt.Println("FetchECBlockByHeight: key=", hex.EncodeToString(key))

	var data []byte
	db.dbLock.Lock()
	data, err = db.lDb.Get(key, db.ro)
	db.dbLock.Unlock()
	if err != nil {
		return nil, err
	}

	ecBlockHash := common.NewHash()
	_, err = ecBlockHash.UnmarshalBinaryData(data)
	if err != nil {
		return nil, err
	}
	fmt.Println("FetchECBlockByHeight: data=", hex.EncodeToString(data), ", hash=", ecBlockHash)
	return db.FetchECBlockByHash(ecBlockHash)
}

// FetchAllECBlocks gets all of the entry credit blocks
func (db *LevelDb) FetchAllECBlocks() (ecBlocks []common.ECBlock, err error) {
	var fromkey = []byte{byte(TBL_CB)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey = []byte{byte(TBL_CB + 1)} // Table Name (1 bytes)
	ecBlockSlice := make([]common.ECBlock, 0, 10)
	var iter iterator.Iterator
	db.dbLock.Lock()
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	db.dbLock.Unlock()

	for iter.Next() {
		ecBlock := common.NewECBlock()
		_, err := ecBlock.UnmarshalBinaryData(iter.Value())
		if err != nil {
			return nil, err
		}
		ecBlockSlice = append(ecBlockSlice, *ecBlock)
	}
	iter.Release()
	err = iter.Error()

	return ecBlockSlice, nil
}
