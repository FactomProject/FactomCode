// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// factomlog is based on github.com/alexcesaro/log and
// github.com/alexcesaro/log/golog (MIT License)

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/util"
	//"github.com/btcsuite/btcd/database"
	"github.com/davecgh/go-spew/spew"
)

var (
	_             = fmt.Print
	cfg           *util.FactomdConfig
	ldbpath       = ""
	db            database.Db // database
	serverPrivKey common.PrivateKey
)

func main() {
	cfg = util.ReadConfig()
	ldbpath = cfg.App.LdbPath
	initDB()

	anchorChainID, _ := common.HexToHash(cfg.Anchor.AnchorChainID)
	fmt.Println("anchorChainID: ", cfg.Anchor.AnchorChainID)

	var err error
	serverPrivKeyHex := cfg.App.ServerPrivKey
	serverPrivKey, err = common.NewPrivateKeyFromHex(serverPrivKeyHex)
	if err != nil {
		panic("Cannot parse Server Private Key from configuration file: " + err.Error())
	}

	processAnchorChain(anchorChainID)
}

func processAnchorChain(anchorChainID *common.Hash) {
	eblocks, _ := db.FetchAllEBlocksByChain(anchorChainID)
	fmt.Println("anchorChain length: ", len(*eblocks))

	for _, eblock := range *eblocks {
		fmt.Printf("anchor chain block=%s\n", spew.Sdump(eblock))

		for _, ebEntry := range eblock.Body.EBEntries {
			entry, _ := db.FetchEntryByHash(ebEntry)

			if entry != nil {
				fmt.Printf("entry=%s\n", spew.Sdump(entry))

				aRecord, err := entryToAnchorRecord(entry)
				if err != nil {
					fmt.Println(err)
				}

				dirBlockInfo, _ := dirBlockInfoToAnchorChain(aRecord)
				err = db.InsertDirBlockInfo(dirBlockInfo)
				if err != nil {
					fmt.Printf("InsertDirBlockInfo error: %s, DirBlockInfo=%s\n", err, spew.Sdump(dirBlockInfo))
				}
			}
		}
	}
}

func dirBlockInfoToAnchorChain(aRecord *anchor.AnchorRecord) (*common.DirBlockInfo, error) {
	dirBlockInfo := new(common.DirBlockInfo)
	dirBlockInfo.DBHeight = aRecord.DBHeight
	txBytes, _ := hex.DecodeString(aRecord.Bitcoin.TXID)
	dirBlockInfo.BTCTxHash, _ = common.NewShaHash(txBytes)
	dirBlockInfo.BTCTxOffset = aRecord.Bitcoin.Offset
	dirBlockInfo.BTCBlockHeight = aRecord.Bitcoin.BlockHeight

	bhBytes, _ := hex.DecodeString(aRecord.Bitcoin.BlockHash)
	dirBlockInfo.BTCBlockHash, _ = common.NewShaHash(bhBytes)
	mrBytes, _ := hex.DecodeString(aRecord.KeyMR)
	dirBlockInfo.DBMerkleRoot, _ = common.NewShaHash(mrBytes)
	dirBlockInfo.BTCConfirmed = true

	dblock, err := db.FetchDBlockByHeight(aRecord.DBHeight)
	if err != nil {
		fmt.Printf("err in FetchDBlockByHeight: %d\n", aRecord.DBHeight)
		//dirBlockInfo.Timestamp = dblock.Header.Timestamp
		dirBlockInfo.DBHash = new(common.Hash)
	} else {
		dirBlockInfo.Timestamp = int64(dblock.Header.Timestamp)
		dirBlockInfo.DBHash = dblock.DBHash
	}
	return dirBlockInfo, nil
}

func entryToAnchorRecord(entry *common.Entry) (aRecord *anchor.AnchorRecord, err error) {
	content := entry.Content
	jsonARecord := content[:(len(content) - 128)]
	jsonSig := content[(len(content) - 128):]

	aRecordSig := serverPrivKey.Sign(jsonARecord)
	newSigString := hex.EncodeToString(aRecordSig.Sig[:])

	if bytes.Compare(jsonSig, []byte(newSigString)) != 0 {
		fmt.Printf("*** anchor chain signature does not match:\n")
		fmt.Printf("original sig:%s\n     new sig:%s", string(jsonSig), newSigString)
	}

	err = json.Unmarshal(jsonARecord, aRecord)
	if err != nil {
		return nil, fmt.Errorf("json.UnMarshall error: %s", err)
	}
	fmt.Printf("entryToAnchorRecord: %s", spew.Sdump(aRecord))

	return aRecord, nil
}

func initDB() {
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	if err != nil {
		fmt.Errorf("err opening db: %v\n", err)
	}

	if db == nil {
		fmt.Println("Creating new db ...")
		db, err = ldb.OpenLevelDB(ldbpath, true)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("Database started from: " + ldbpath)
}
