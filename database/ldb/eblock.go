package ldb

import (
	"encoding/binary"
	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"log"

	"github.com/FactomProject/goleveldb/leveldb/util"
)

// ProcessEBlockBatche inserts the EBlock and update all it's ebentries in DB
func (db *LevelDb) ProcessEBlockBatch(eblock *common.EBlock) error {

	if eblock != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		if len(eblock.EBEntries) < 1 {
			return errors.New("Empty eblock!")
		}

		binaryEblock, err := eblock.MarshalBinary()
		if err != nil {
			return err
		}
		
		if eblock.EBHash == nil {
			eblock.EBHash = common.Sha(binaryEblock)
		}

		if eblock.MerkleRoot == nil {
			eblock.BuildMerkleRoot()
		}
		
		// Insert the binary entry block
		var key []byte = []byte{byte(TBL_EB)}
		key = append(key, eblock.EBHash.Bytes()...)
		db.lbatch.Put(key, binaryEblock)

		// Insert the entry block merkle root cross reference
		key = []byte{byte(TBL_EB_MR)}
		key = append(key, eblock.MerkleRoot.Bytes()...)
		binaryEBHash, _ := eblock.EBHash.MarshalBinary()
		db.lbatch.Put(key, binaryEBHash)

		// Insert the entry block number cross reference
		key = []byte{byte(TBL_EB_CHAIN_NUM)}
		key = append(key, eblock.Header.ChainID.Bytes()...)
		bytes := make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, eblock.Header.EBHeight)
		key = append(key, bytes...)
		db.lbatch.Put(key, binaryEBHash)

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}

// FetchEBlockByMR gets an entry block by merkle root from the database.
func (db *LevelDb) FetchEBlockByMR(eBMR *common.Hash) (eBlock *common.EBlock, err error) {
	eBlockHash, _ := db.FetchEBHashByMR(eBMR)

	if eBlockHash != nil {
		eBlock, _ = db.FetchEBlockByHash(eBlockHash)
	}

	return eBlock, nil
}

// FetchEntryBlock gets an entry by hash from the database.
func (db *LevelDb) FetchEBlockByHash(eBlockHash *common.Hash) (eBlock *common.EBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_EB)}
	key = append(key, eBlockHash.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		eBlock = new(common.EBlock)
		eBlock.UnmarshalBinary(data)
	}
	return eBlock, nil
}

// FetchEBlockByHeight gets an entry block by height from the database.
// Need to rewrite since only the cross ref is stored in db ??
/*func (db *LevelDb) FetchEBlockByHeight(chainID * common.Hash, eBlockHeight uint32) (eBlock *common.EBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_EB_CHAIN_NUM)}
	key = append(key, chainID.Bytes...)
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, eBlockHeight)
	key = append(key, bytes...)	
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		eBlock = new(common.EBlock)
		eBlock.UnmarshalBinary(data)
	}
	return eBlock, nil
}
*/

// FetchEBHashByMR gets an entry by hash from the database.
func (db *LevelDb) FetchEBHashByMR(eBMR *common.Hash) (eBlockHash *common.Hash, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_EB_MR)}
	key = append(key, eBMR.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		eBlockHash = new(common.Hash)
		eBlockHash.UnmarshalBinary(data)
	}
	return eBlockHash, nil
}

// InsertChain inserts the newly created chain into db
func (db *LevelDb) InsertChain(chain *common.EChain) (err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	defer db.lbatch.Reset()

	binaryChain, _ := chain.MarshalBinary()

	var chainByHashKey []byte = []byte{byte(TBL_CHAIN_HASH)}
	chainByHashKey = append(chainByHashKey, chain.ChainID.Bytes()...)

	db.lbatch.Put(chainByHashKey, binaryChain)

	err = db.lDb.Write(db.lbatch, db.wo)
	if err != nil {
		log.Println("batch failed %v\n", err)
		return err
	}

	return nil
}

// FetchChainByHash gets a chain by chainID
func (db *LevelDb) FetchChainByHash(chainID *common.Hash) (chain *common.EChain, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var key []byte = []byte{byte(TBL_CHAIN_HASH)}
	key = append(key, chainID.Bytes()...)
	data, err := db.lDb.Get(key, db.ro)

	if data != nil {
		chain = new(common.EChain)
		chain.UnmarshalBinary(data)
	}
	return chain, nil
}

// FetchAllChains get all of the cahins
func (db *LevelDb) FetchAllChains() (chains []common.EChain, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_CHAIN_HASH)}   // Table Name (1 bytes)
	var tokey []byte = []byte{byte(TBL_CHAIN_HASH + 1)} // Table Name (1 bytes)

	chainSlice := make([]common.EChain, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)
	for iter.Next() {
		var chain common.EChain
		chain.UnmarshalBinary(iter.Value())
		chainSlice = append(chainSlice, chain)
	}
	iter.Release()
	err = iter.Error()

	return chainSlice, err
}

// FetchAllEBlocksByChain gets all of the blocks by chain id
func (db *LevelDb) FetchAllEBlocksByChain(chainID *common.Hash) (eBlocks *[]common.EBlock, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	var fromkey []byte = []byte{byte(TBL_EB_CHAIN_NUM)} // Table Name (1 bytes)
	fromkey = append(fromkey, chainID.Bytes()...) // Chain Type (32 bytes)
	var tokey []byte = addOneToByteArray(fromkey)

	eBlockSlice := make([]common.EBlock, 0, 10)

	iter := db.lDb.NewIterator(&util.Range{Start: fromkey, Limit: tokey}, db.ro)

	for iter.Next() {
		eBlockHash := new(common.Hash)
		eBlockHash.UnmarshalBinary(iter.Value())

		var key []byte = []byte{byte(TBL_EB)}
		key = append(key, eBlockHash.Bytes()...)
		data, _ := db.lDb.Get(key, db.ro)

		if data != nil {
			eBlock := new(common.EBlock)
			eBlock.UnmarshalBinary(data)
			eBlock.EBHash = eBlockHash
			eBlockSlice = append(eBlockSlice, *eBlock)
		}
	}
	iter.Release()
	err = iter.Error()

	return &eBlockSlice, nil
}

// Internal db use only
func addOneToByteArray(input []byte) (output []byte) {
	if input == nil {
		return []byte{byte(1)}
	}
	output = make([]byte, len(input))
	copy(output, input)
	for i := len(input); i > 0; i-- {
		if output[i-1] <= 255 {
			output[i-1] = output[i-1] + 1
			break
		}
	}
	return output
}
