package main

import (
	"bytes"
	"fmt"
	"testing"

)
 

func TestDblock(t *testing.T) {
	fmt.Println("\nTest dblock===========================================================================")
	
	initDB()
	
	dblock, _ := db.FetchDBlockByHeight(0)
	
	dblock.BuildKeyMerkleRoot()
	
	dblock2, _ := db.FetchDBlockByMR(dblock.KeyMR)


	if dblock2 == nil {
		t.Errorf("Invalid output: dblock2 not found")
	}
	bytes1, _ := dblock.MarshalBinary()
	bytes2, _ := dblock2.MarshalBinary()

	if bytes.Compare(bytes1, bytes2) != 0 {
		t.Errorf("Invalid output")
	}
}

