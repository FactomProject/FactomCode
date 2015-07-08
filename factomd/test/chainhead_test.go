package main

import (
	//	"bytes"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"testing"
	//	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/common"
)

func TestDblock(t *testing.T) {
	fmt.Println("\nTest chain head===========================================================================")

	initDB()

	// directory block ------------------
	dchainid := new(common.Hash)
	dchainid.SetBytes(common.D_CHAINID)

	dbMR, _ := db.FetchHeadMRByChainID(dchainid)

	dblk, _ := db.FetchDBlockByMR(dbMR)

	fmt.Printf("dblk=%s\n", spew.Sdump(dblk))

	// admin block ------------------------
	achainid := new(common.Hash)
	achainid.SetBytes(common.ADMIN_CHAINID)

	abMR, _ := db.FetchHeadMRByChainID(achainid)

	ablk, _ := db.FetchABlockByHash(abMR)

	fmt.Printf("ablk=%s\n", spew.Sdump(ablk))

	//EC block ------------------------------------
	ecchainid := new(common.Hash)
	ecchainid.SetBytes(common.EC_CHAINID)

	ecbMR, _ := db.FetchHeadMRByChainID(ecchainid)

	ecblk, _ := db.FetchECBlockByHash(ecbMR)

	fmt.Printf("ecblk=%s\n", spew.Sdump(ecblk))

	//factoid block ------------------------------------
	fchainid := new(common.Hash)
	fchainid.SetBytes(common.FACTOID_CHAINID)

	fbMR, _ := db.FetchHeadMRByChainID(fchainid)

	fblk, _ := db.FetchFBlockByHash(fbMR)

	fmt.Printf("fblk=%s\n", spew.Sdump(fblk))

}
