package ldb

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid/block"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/iterator"
	"github.com/FactomProject/goleveldb/leveldb/util"
)

// ProcessFBlockBatch inserts the factoid block
func (db *LevelDb) ProcessFBlockBatch(block block.IFBlock) error {
	if block == nil {
		return nil
	}
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	err := db.ProcessFBlockMultiBatch(block)
	if err != nil {
		return err
	}

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		fmt.Printf("batch failed %v\n", err)
		return err
	}
	return nil
}

func (db *LevelDb) ProcessFBlockMultiBatch(block block.IFBlock) error {
	if block == nil {
		return nil
	}

	if db.lbatch == nil {
		return fmt.Errorf("db.lbatch == nil")
	}

	binaryBlock, err := block.MarshalBinary()
	if err != nil {
		return err
	}

	scHash := block.GetHash()

	// Insert the binary factom block
	var key = []byte{byte(TBL_SC)}
	key = append(key, scHash.Bytes()...)
	db.lbatch.Put(key, binaryBlock)

	// Insert the sc block number cross reference
	key = []byte{byte(TBL_SC_NUM)}
	key = append(key, common.FACTOID_CHAINID...)
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, block.GetDBHeight())
	key = append(key, bytes...)
	db.lbatch.Put(key, scHash.Bytes())

	// Update the chain head reference
	key = []byte{byte(TBL_CHAIN_HEAD)}
	key = append(key, common.FACTOID_CHAINID...)
	db.lbatch.Put(key, scHash.Bytes())

	return nil
}

// FetchFBlockByHash gets an factoid block by hash from the database.
func (db *LevelDb) FetchFBlockByHash(hash *common.Hash) (FBlock block.IFBlock, err error) {
	var key = []byte{byte(TBL_SC)}
	key = append(key, hash.Bytes()...)
	var data []byte
	db.dbLock.Lock()
	data, err = db.lDb.Get(key, db.ro)
	db.dbLock.Unlock()

	if data != nil {
		FBlock = new(block.FBlock)
		_, err := FBlock.UnmarshalBinaryData(data)
		if err != nil {
			return nil, err
		}
	}
	return FBlock, nil
}

// FetchFBlockByHeight gets an factoid block by hash from the database.
func (db *LevelDb) FetchFBlockByHeight(height uint32) (block.IFBlock, error) {
	var key = []byte{byte(TBL_SC_NUM)}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, height)
	key = append(key, common.FACTOID_CHAINID...)
	key = append(key, buf.Bytes()...)

	var data []byte
	var err error
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
	return db.FetchFBlockByHash(ecBlockHash)
}

// FetchAllFBlocks gets all of the factoid blocks
func (db *LevelDb) FetchAllFBlocks() (FBlocks []block.IFBlock, err error) {
	var fromkey = []byte{byte(TBL_SC)}   // Table Name (1 bytes)						// Timestamp  (8 bytes)
	var tokey = []byte{byte(TBL_SC + 1)} // Table Name (1 bytes)
	var iter iterator.Iterator
	FBlockSlice := make([]block.IFBlock, 0, 10)
	db.dbLock.Lock()
	iter = db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	db.dbLock.Unlock()

	for iter.Next() {
		FBlock := new(block.FBlock)
		_, err := FBlock.UnmarshalBinaryData(iter.Value())
		if err != nil {
			return nil, err
		}

		FBlockSlice = append(FBlockSlice, FBlock)

	}
	iter.Release()
	err = iter.Error()

	return FBlockSlice, nil
}
