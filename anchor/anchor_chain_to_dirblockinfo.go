package main

import (
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/util"
	"github.com/btcsuite/btcd/database"
	"github.com/davecgh/go-spew/spew"
)

var (
	_       = fmt.Print
	cfg     *util.FactomdConfig
	ldbpath = ""
	db      database.Db // database

)

func main() {
	cfg = util.ReadConfig()
	ldbpath = cfg.App.LdbPath

	anchorChainID, _ := common.HexToHash(cfg.Anchor.AnchorChainID)
	fmt.Println("anchorChainID: ", anchorChainID)

	eblocks, _ := db.FetchAllEBlocksByChain(anchorChainID)
	fmt.Println("anchorChain length: ", len(*eblocks))

	for _, eblock := range *eblocks {
		fmt.Printf("anchor chain block=%s\n", spew.Sdump(eblock))
		for _, ebEntry := range eblock.Body.EBEntries {
			entry, _ := db.FetchEntryByHash(ebEntry)
			if entry != nil {
				fmt.Printf("entry=%s\n", spew.Sdump(entry))
				content := entry.Content
			}
		}
	}
}

func initDB() {
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	if err != nil {
		fmt.Errorf("err opening db: %v\n", err)
	}

	if db == nil {
		fmt.Info("Creating new db ...")
		db, err = ldb.OpenLevelDB(ldbpath, true)
		if err != nil {
			panic(err)
		}
	}
	fmt.Info("Database started from: " + ldbpath)
}
