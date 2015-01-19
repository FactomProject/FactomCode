package ldb

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/cache"
	"github.com/FactomProject/goleveldb/leveldb/opt"

	"github.com/btcsuite/btcwire"
)

const (
	dbVersion     int = 2
	dbMaxTransCnt     = 20000
	dbMaxTransMem     = 64 * 1024 * 1024 // 64 MB
)

// the "table" prefix
const (
	TBL_ENTRY uint8 = iota
	TBL_ENTRY_QUEUE
	TBL_ENTRY_INFO

	TBL_EB       //3
	TBL_EB_QUEUE //4
	TBL_EB_INFO  //5
	TBL_EB_CHAIN_NUM
	TBL_EB_MR

	TBL_DB //8
	TBL_DB_NUM
	TBL_DBATCH
	TBL_DB_BATCH

	TBL_CHAIN_HASH //12
	TBL_CHAIN_NAME

	TBL_CB //14
	TBL_CB_NUM
	TBL_CB_INFO
)

// the process status in db
const (
	STATUS_IN_QUEUE uint8 = iota
	STATUS_PROCESSED
)

// chain type key prefix ??
var currentChainType uint32 = 1

var isLookupDB bool = true // to be put in property file

type tTxInsertData struct {
	txsha   *btcwire.ShaHash
	blockid int64
	txoff   int
	txlen   int
	usedbuf []byte
}

type LevelDb struct {
	// lock preventing multiple entry
	dbLock sync.Mutex

	// leveldb pieces
	lDb *leveldb.DB
	ro  *opt.ReadOptions
	wo  *opt.WriteOptions

	lbatch *leveldb.Batch

	nextBlock int64

	lastBlkShaCached bool
	lastBlkSha       btcwire.ShaHash
	lastBlkIdx       int64

	//	txUpdateMap      map[btcwire.ShaHash]*txUpdateObj
	//	txSpentUpdateMap map[btcwire.ShaHash]*spentTxUpdate
}

var CurrentDBVersion int32 = 1

//to be removed??
func OpenLevelDB(dbpath string, create bool) (pbdb database.Db, err error) {
	return openDB(dbpath, create)
}

func openDB(dbpath string, create bool) (pbdb database.Db, err error) {
	var db LevelDb
	var tlDb *leveldb.DB
	var dbversion int32

	defer func() {
		if err == nil {
			db.lDb = tlDb

			//			db.txUpdateMap = map[btcwire.ShaHash]*txUpdateObj{}
			//			db.txSpentUpdateMap = make(map[btcwire.ShaHash]*spentTxUpdate)

			pbdb = &db
		}
	}()

	if create == true {
		err = os.MkdirAll(dbpath, 0750)
		if err != nil {
			log.Println("mkdir failed %v %v", dbpath, err)
			return
		}
	} else {
		_, err = os.Stat(dbpath)
		if err != nil {
			return
		}
	}

	needVersionFile := false
	verfile := dbpath + ".ver"
	fi, ferr := os.Open(verfile)
	if ferr == nil {
		defer fi.Close()

		ferr = binary.Read(fi, binary.BigEndian, &dbversion)
		if ferr != nil {
			dbversion = ^0
		}
	} else {
		if create == true {
			needVersionFile = true
			dbversion = CurrentDBVersion
		}
	}

	myCache := cache.NewEmptyCache()
	opts := &opt.Options{
		BlockCache:   myCache,
		MaxOpenFiles: 256,
		Compression:  opt.NoCompression,
	}

	switch dbversion {
	case 0:
		opts = &opt.Options{}
	case 1:
		// uses defaults from above
	default:
		err = fmt.Errorf("unsupported db version %v", dbversion)
		return
	}

	tlDb, err = leveldb.OpenFile(dbpath, opts)
	if err != nil {
		return
	}

	// If we opened the database successfully on 'create'
	// update the
	if needVersionFile {
		fo, ferr := os.Create(verfile)
		if ferr != nil {
			// TODO(design) close and delete database?
			err = ferr
			return
		}
		defer fo.Close()
		err = binary.Write(fo, binary.BigEndian, dbversion)
		if err != nil {
			return
		}
	}

	return
}

func (db *LevelDb) close() error {
	return db.lDb.Close()
}

// Sync verifies that the database is coherent on disk,
// and no outstanding transactions are in flight.
func (db *LevelDb) Sync() error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	// while specified by the API, does nothing
	// however does grab lock to verify it does not return until other operations are complete.
	return nil
}

// Close cleanly shuts down database, syncing all data.
func (db *LevelDb) Close() error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	return db.close()
}

func int64ToKey(keyint int64) []byte {
	key := strconv.FormatInt(keyint, 10)
	return []byte(key)
}

func shaBlkToKey(sha *btcwire.ShaHash) []byte {
	shaB := sha.Bytes()
	return shaB
}

func shaTxToKey(sha *btcwire.ShaHash) []byte {
	shaB := sha.Bytes()
	shaB = append(shaB, "tx"...)
	return shaB
}

func shaSpentTxToKey(sha *btcwire.ShaHash) []byte {
	shaB := sha.Bytes()
	shaB = append(shaB, "sx"...)
	return shaB
}

func (db *LevelDb) lBatch() *leveldb.Batch {
	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	return db.lbatch
}

func (db *LevelDb) RollbackClose() error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	return db.close()
}
