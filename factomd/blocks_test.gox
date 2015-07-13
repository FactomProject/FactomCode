package main

import (
	//	"bytes"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/davecgh/go-spew/spew"
	"testing"
)

var dirBlockHeight uint32 = 0

func TestBlocks(t *testing.T) {
	fmt.Println("\nTest Blocks===========================================================================")

	loadConfigurations()
	
	initDB()

	// directory block ------------------
	dblk, _ := db.FetchDBlockByHeight(dirBlockHeight)
	t.Logf("dblk=%s\n", spew.Sdump(dblk))

	// admin chain ------------------------
	achainid := new(common.Hash)
	achainid.SetBytes(common.ADMIN_CHAINID)

	//EC chain ------------------------------------
	ecchainid := new(common.Hash)
	ecchainid.SetBytes(common.EC_CHAINID)

	//factoid chain ------------------------------------
	fchainid := new(common.Hash)
	fchainid.SetBytes(common.FACTOID_CHAINID)

	for _, dbEntry := range dblk.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchainid.String():
			ecblk, _ := db.FetchECBlockByHash(dbEntry.KeyMR)
			t.Logf("ecblk=%s\n", spew.Sdump(ecblk))
		case achainid.String():
			ablk, _ := db.FetchABlockByHash(dbEntry.KeyMR)
			t.Logf("ablk=%s\n", spew.Sdump(ablk))
		case fchainid.String():
			fblk, _ := db.FetchFBlockByHash(dbEntry.KeyMR)
			t.Logf("fblk=%s\n", spew.Sdump(fblk))
		default:
			eBlk, _ := db.FetchEBlockByMR(dbEntry.KeyMR)
			// validate every entry in EBlock
			for _, ebEntry := range eBlk.EBEntries {
				// continue if the entry arleady exists in db
				entry, _ := db.FetchEntryByHash(ebEntry.EntryHash)
				t.Logf("entryHash=%s", spew.Sdump(ebEntry.EntryHash))
				t.Logf("entry=%s\n", spew.Sdump(entry))
			}

		}
	}

}
