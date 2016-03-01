// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// factomlog is based on github.com/alexcesaro/log and
// github.com/alexcesaro/log/golog (MIT License)

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/util"
	"github.com/btcsuitereleases/btcd/wire"
	"github.com/FactomProject/go-spew/spew"
)

var (
	_               = fmt.Print
	cfg             *util.FactomdConfig
	db              database.Db
	anchorChainID   *common.Hash
	dirBlockInfoMap map[uint32]*common.DirBlockInfo //DBHeight as key
	dblocks         []common.DirectoryBlock
	eblocks         *[]common.EBlock
)

// This utility is used to convert client-side factomd database to server-side
// database.
// It uses the factomd config file in default location: /home/.factom/factomd.conf
// and the default database specified in this config: /home/.factom/ldb
// You should copy the client-side db to this location, and then go run this util.
//
func main() {

	sanityCheck()

	processAnchorChain()

	createMissingDirBlockInfo()

	fmt.Println("done!")

	//initDB("/home/bw/.factom/ldb.prd")
	//dirBlockInfoMap, _ := db.FetchAllDirBlockInfo() // map[string]*common.DirBlockInfo
	//for _, dirBlockInfo := range dirBlockInfoMap {
	//fmt.Printf("dirBlockInfo: %s\n", spew.Sdump(dirBlockInfo))
	//}
}

func sanityCheck() {
	cfg = util.ReadConfig()
	ldbpath := cfg.App.LdbPath
	initDB(ldbpath)
	dirBlockInfoMap = make(map[uint32]*common.DirBlockInfo)
	anchorChainID, _ = common.HexToHash(cfg.Anchor.AnchorChainID)
	fmt.Printf("ldbPath=%s, anchorChainID=%s\n", ldbpath, cfg.Anchor.AnchorChainID)

	dblocks, _ = db.FetchAllDBlocks()
	dirBlockInfoMap, _ := db.FetchAllDirBlockInfo()
	eblocks, _ = db.FetchAllEBlocksByChain(anchorChainID)
	fmt.Printf("There are %d directory blocks, %d DirBlockInfos, and %d anchor chain blocks in this database.\n",
		len(dblocks), len(dirBlockInfoMap), len(*eblocks))

	if len(dblocks) == len(dirBlockInfoMap) {
		fmt.Println("All dir blocks have dirBlockInfo. All good and done!")
		os.Exit(0)
	}
}

func createMissingDirBlockInfo() {
	fmt.Println("create DirBlockInfo for those un-anchored DirBlocks")
	//dblocks, _ := db.FetchAllDBlocks()
	for _, dblock := range dblocks {
		if _, ok := dirBlockInfoMap[dblock.Header.DBHeight]; ok {
			continue
		} else {
			dblock.BuildKeyMerkleRoot()
			//fmt.Printf("creating missing dirBlockInfo for dir block=%s\n", spew.Sdump(dblock))
			dirBlockInfo := common.NewDirBlockInfoFromDBlock(&dblock)
			//fmt.Printf("creating missing dirBlockInfo. DirBlockInfo=%s\n", spew.Sdump(dirBlockInfo))
			err := db.InsertDirBlockInfo(dirBlockInfo)
			if err != nil {
				fmt.Printf("InsertDirBlockInfo error: %s, DirBlockInfo=%s\n", err, spew.Sdump(dirBlockInfo))
			}
			dirBlockInfoMap[dirBlockInfo.DBHeight] = dirBlockInfo
		}
	}
}

func processAnchorChain() {
	fmt.Println("processAnchorChain")
	//eblocks, _ := db.FetchAllEBlocksByChain(anchorChainID)
	//fmt.Println("anchorChain length: ", len(*eblocks))
	for _, eblock := range *eblocks {
		//fmt.Printf("anchor chain block=%s\n", spew.Sdump(eblock))
		if eblock.Header.EBSequence == 0 {
			continue
		}
		for _, ebEntry := range eblock.Body.EBEntries {
			entry, _ := db.FetchEntryByHash(ebEntry)
			if entry != nil {
				//fmt.Printf("entry=%s\n", spew.Sdump(entry))
				aRecord, err := entryToAnchorRecord(entry)
				if err != nil {
					fmt.Println(err)
				}
				dirBlockInfo, _ := anchorChainToDirBlockInfo(aRecord)
				err = db.InsertDirBlockInfo(dirBlockInfo)
				if err != nil {
					fmt.Printf("InsertDirBlockInfo error: %s, DirBlockInfo=%s\n", err, spew.Sdump(dirBlockInfo))
				}
				dirBlockInfoMap[dirBlockInfo.DBHeight] = dirBlockInfo
			}
		}
	}
}

func anchorChainToDirBlockInfo(aRecord *anchor.AnchorRecord) (*common.DirBlockInfo, error) {
	dirBlockInfo := new(common.DirBlockInfo)
	dirBlockInfo.DBHeight = aRecord.DBHeight
	dirBlockInfo.BTCTxOffset = aRecord.Bitcoin.Offset
	dirBlockInfo.BTCBlockHeight = aRecord.Bitcoin.BlockHeight
	mrBytes, _ := hex.DecodeString(aRecord.KeyMR)
	dirBlockInfo.DBMerkleRoot, _ = common.NewShaHash(mrBytes)
	dirBlockInfo.BTCConfirmed = true

	txSha, _ := wire.NewShaHashFromStr(aRecord.Bitcoin.TXID)
	dirBlockInfo.BTCTxHash = toHash(txSha)
	blkSha, _ := wire.NewShaHashFromStr(aRecord.Bitcoin.BlockHash)
	dirBlockInfo.BTCBlockHash = toHash(blkSha)

	dblock, err := db.FetchDBlockByHeight(aRecord.DBHeight)
	if err != nil {
		fmt.Printf("err in FetchDBlockByHeight: %d\n", aRecord.DBHeight)
		dirBlockInfo.DBHash = new(common.Hash)
	} else {
		dirBlockInfo.Timestamp = int64(dblock.Header.Timestamp * 60)
		dirBlockInfo.DBHash = dblock.DBHash
	}
	//fmt.Printf("dirBlockInfo: %s\n", spew.Sdump(dirBlockInfo))
	return dirBlockInfo, nil
}

func entryToAnchorRecord(entry *common.Entry) (*anchor.AnchorRecord, error) {
	content := entry.Content
	jsonARecord := content[:(len(content) - 128)]
	jsonSigBytes := content[(len(content) - 128):]
	jsonSig, err := hex.DecodeString(string(jsonSigBytes))
	if err != nil {
		fmt.Printf("*** hex.Decode jsonSigBytes error: %s\n", err.Error())
	}

	//fmt.Println("bytes decoded: ", hex.DecodedLen(len(jsonSigBytes)))
	//fmt.Printf("jsonARecord: %s\n", string(jsonARecord))
	//fmt.Printf("    jsonSig: %s\n", string(jsonSigBytes))

	pubKeySlice := make([]byte, 32, 32)
	pubKey := common.PubKeyFromString(common.SERVER_PUB_KEY)
	copy(pubKeySlice, pubKey.Key[:])
	verified := common.VerifySlice(pubKeySlice, jsonARecord, jsonSig)

	if !verified {
		fmt.Printf("*** anchor chain signature does NOT match:\n")
	} else {
		//fmt.Printf("&&& anchor chain signature does MATCH:\n")
	}

	aRecord := new(anchor.AnchorRecord)
	err = json.Unmarshal(jsonARecord, aRecord)
	if err != nil {
		return nil, fmt.Errorf("json.UnMarshall error: %s", err)
	}
	//fmt.Printf("entryToAnchorRecord: %s", spew.Sdump(aRecord))

	return aRecord, nil
}

func initDB(ldbpath string) {
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

func toHash(txHash *wire.ShaHash) *common.Hash {
	h := new(common.Hash)
	h.SetBytes(txHash.Bytes())
	return h
}
