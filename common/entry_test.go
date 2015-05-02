package common

import (
	"bytes"
	"fmt"
	"testing"
)

func TestFirstEntry(t *testing.T) {
	fmt.Println("\nTestFirstEntry===========================================================================")

	entry := new(Entry)

	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1asdfadfasdf"))
	entry.ExtIDs = append(entry.ExtIDs, []byte(""))
	entry.ExtIDs = append(entry.ExtIDs, []byte("3"))
	entry.ChainID, _ = entry.GenerateIDFromName()

	entry.Data = []byte("1asdf asfas dfsg\"08908098(*)*^*&%&%&$^#%##%$$@$@#$!$#!$#@!~@!#@!%#@^$#^&$*%())_+_*^*&^&\"\"?>?<<>/./,")

	bytes1, err := entry.MarshalBinary()
	fmt.Printf("bytes1:%v\n", bytes1)

	entry2 := new(Entry)
	entry2.UnmarshalBinary(bytes1)

	bytes2, _ := entry2.MarshalBinary()
	fmt.Printf("bytes2:%v\n", bytes2)

	if bytes.Compare(bytes1, bytes2) != 0 {
		t.Errorf("Invalid output")
	}

	if err != nil {
		t.Errorf("Error:%v", err)
	}
}

func TestEntry(t *testing.T) {
	fmt.Println("\nTestEntry===========================================================================")

	entry := new(Entry)

	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1asdfadfasdf"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("2asdfas asfasfasfafas "))
	entry.ExtIDs = append(entry.ExtIDs, []byte("3sd fasfas fsaf asf asfasfsafsfa"))
	entry.ChainID = new(Hash)
	entry.ChainID.Bytes = EC_CHAINID
	entry.Data = []byte("1asdf asfas fasfadfasdfasfdfff12345")

	bytes1, err := entry.MarshalBinary()
	fmt.Printf("bytes1:%v\n", bytes1)

	entry2 := new(Entry)
	entry2.UnmarshalBinary(bytes1)

	bytes2, _ := entry2.MarshalBinary()
	fmt.Printf("bytes2:%v\n", bytes2)

	if bytes.Compare(bytes1, bytes2) != 0 {
		t.Errorf("Invalid output")
	}

	if err != nil {
		t.Errorf("Error:%v", err)
	}
}
