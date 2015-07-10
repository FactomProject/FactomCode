package common_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)

func TestUnmarshal(t *testing.T) {
	fmt.Printf("TestNewUnmarshal\n---\n")
	e := new(Entry)

	data, err := hex.DecodeString("00954d5a49fd70d9b8bcdb35d252267829957f7ef7fa6c74f88419bdc5e82209f4000600110004746573745061796c6f616448657265")
	if err != nil {
		t.Error(err)
	}

	if err := e.UnmarshalBinary(data); err != nil {
		t.Error(err)
	}

	fmt.Println(e)
}

func TestFirstEntry(t *testing.T) {
	fmt.Println("\nTestFirstEntry===========================================================================")

	entry := new(Entry)

	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1asdfadfasdf"))
	entry.ExtIDs = append(entry.ExtIDs, []byte(""))
	entry.ExtIDs = append(entry.ExtIDs, []byte("3"))
	entry.ChainID = new(Hash)
	err := entry.ChainID.SetBytes(EC_CHAINID)
	if err != nil {
		t.Errorf("Error:%v", err)
	}

	entry.Content = []byte("1asdf asfas dfsg\"08908098(*)*^*&%&%&$^#%##%$$@$@#$!$#!$#@!~@!#@!%#@^$#^&$*%())_+_*^*&^&\"\"?>?<<>/./,")

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
		t.Errorf("Error: %v", err)
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
	err := entry.ChainID.SetBytes(EC_CHAINID)
	if err != nil {
		t.Errorf("Error:%v", err)
	}

	entry.Content = []byte("1asdf asfas fasfadfasdfasfdfff12345")

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
