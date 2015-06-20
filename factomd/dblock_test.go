package main

import (
	"bytes"
	"fmt"
	"testing"
	"github.com/davecgh/go-spew/spew"	

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

func TestDblock2(t *testing.T) {
	fmt.Println("\nTest dblock2===========================================================================")
	
	list, _ := db.FetchHeightRange(0, 5)
	fmt.Printf("TestDblock2: list=%s\n", spew.Sdump(list))	
	
	
	height,_ := db.FetchBlockHeightBySha(&list[0])

	fmt.Printf("height=%s\n", spew.Sdump(height))	
	
}

