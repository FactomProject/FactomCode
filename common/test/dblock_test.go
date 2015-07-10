package common_test

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)

func TestDblock(t *testing.T) {
	fmt.Println("\nTest dblock===========================================================================")

	dblock := new(DirectoryBlock)

	dblock.Header = new(DBlockHeader)
	dblock.Header.DBHeight = 1
	dblock.Header.BodyMR = NewHash()
	dblock.Header.BlockCount = 0
	dblock.Header.NetworkID = 9
	dblock.Header.PrevFullHash = NewHash()
	dblock.Header.PrevKeyMR = NewHash()
	dblock.Header.Timestamp = 1234
	dblock.Header.Version = 1

	de := new(DBEntry)
	de.ChainID = NewHash()
	de.KeyMR = NewHash()

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
