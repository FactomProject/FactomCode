package common

import (
	"bytes"
	"fmt"
	"testing"
)

func TestDblock(t *testing.T) {
	fmt.Println("\nTest dblock===========================================================================")

	dblock := new(DirectoryBlock)

	dblock.Header = new(DBlockHeader)
	dblock.Header.BlockHeight = 1
	dblock.Header.BodyMR = NewHash()
	dblock.Header.EntryCount =0
	dblock.Header.NetworkID = 9
	dblock.Header.PrevBlockHash = NewHash()
	dblock.Header.PrevKeyMR = NewHash()
	dblock.Header.StartTime = 1234
	dblock.Header.Version = 1
	
	de := new(DBEntry)
	de.ChainID = NewHash()
	de.MerkleRoot = NewHash()
	
	dblock.DBEntries = make([]*DBEntry, 0, 5)
	dblock.DBEntries = append(dblock.DBEntries, de)

	bytes1, err := dblock.Header.MarshalBinary()
	fmt.Printf("bytes1:%v\n", bytes1)

	dblock2 := new(DirectoryBlock)
	dblock2.Header = new(DBlockHeader)
	dblock2.Header.UnmarshalBinary(bytes1)

	bytes2, _ := dblock2.Header.MarshalBinary()
	fmt.Printf("bytes2:%v\n", bytes2)

	if bytes.Compare(bytes1, bytes2) != 0 {
		t.Errorf("Invalid output")
	}

	if err != nil {
		t.Errorf("Error:%v", err)
	}
}
