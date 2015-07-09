package main

import (
	//	"bytes"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"testing"
	"github.com/FactomProject/FactomCode/common"
)

var dirBlockHeight uint32 = 241

func TestBlocks(t *testing.T) {
	fmt.Println("\nTest Blocks===========================================================================")

	initDB()
	
	// directory block ------------------	
	dblk, _ := db.FetchDBlockByHeight(dirBlockHeight)
	fmt.Printf("dblk=%s\n", spew.Sdump(dblk))

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
			ecblk, _ := db.FetchECBlockByHash(dbEntry.MerkleRoot)
			fmt.Printf("ecblk=%s\n", spew.Sdump(ecblk))
		case achainid.String():
			ablk, _ := db.FetchABlockByHash(dbEntry.MerkleRoot)
			fmt.Printf("ablk=%s\n", spew.Sdump(ablk))
		case fchainid.String():
			fblk, _ := db.FetchFBlockByHash(dbEntry.MerkleRoot)
			fmt.Printf("fblk=%s\n", spew.Sdump(fblk))
		default:
			eBlk, _ := db.FetchEBlockByMR(dbEntry.MerkleRoot)
				// validate every entry in EBlock
				for _, ebEntry := range eBlk.EBEntries {
					// continue if the entry arleady exists in db
					entry, _ := db.FetchEntryByHash(ebEntry.EntryHash)
					fmt.Printf("entryHash=%s", spew.Sdump(ebEntry.EntryHash))					
					fmt.Printf("entry=%s\n", spew.Sdump(entry))
				}
			
		}
	}	

}
