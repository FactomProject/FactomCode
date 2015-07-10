package common_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)

func TestABlockHeaderMarshalUnmarshal(t *testing.T) {
	fmt.Printf("---\nTestABlockHeaderMarshalUnmarshal\n---\n")

	header := createTestAdminHeader()

	binary, err := header.MarshalBinary()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	header2 := new(ABlockHeader)
	err = header2.UnmarshalBinary(binary)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if bytes.Compare(header.AdminChainID.Bytes(), header2.AdminChainID.Bytes()) != 0 {
		t.Error("AdminChainIDs are not identical")
	}

	if bytes.Compare(header.PrevFullHash.Bytes(), header2.PrevFullHash.Bytes()) != 0 {
		t.Error("PrevFullHashes are not identical")
	}

	if header.DBHeight != header2.DBHeight {
		t.Error("DBHeights are not identical")
	}

	if header.HeaderExpansionSize != header2.HeaderExpansionSize {
		t.Error("HeaderExpansionSizes are not identical")
	}

	if bytes.Compare(header.HeaderExpansionArea, header2.HeaderExpansionArea) != 0 {
		t.Error("HeaderExpansionAreas are not identical")
	}

	if header.MessageCount != header2.MessageCount {
		t.Error("HeaderExpansionSizes are not identical")
	}

	if header.BodySize != header2.BodySize {
		t.Error("HeaderExpansionSizes are not identical")
	}

}

func TestMarshalledSize(t *testing.T) {
	fmt.Printf("---\nTestMarshalledSize\n---\n")

	header := createTestAdminHeader()
	marshalled, err := header.MarshalBinary()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if header.MarshalledSize() != uint64(len(marshalled)) {
		t.Error("Predicted size does not match actual size")
	}

	header2 := createSmallTestAdminHeader()
	marshalled2, err := header2.MarshalBinary()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if header2.MarshalledSize() != uint64(len(marshalled2)) {
		t.Error("Predicted size does not match actual size")
	}
}

func createTestAdminHeader() *ABlockHeader {
	header := new(ABlockHeader)

	p, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash, _ := NewShaHash(p)
	header.AdminChainID = hash
	p, _ = hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	hash, _ = NewShaHash(p)
	header.PrevFullHash = hash
	header.DBHeight = 123

	header.HeaderExpansionSize = 5
	header.HeaderExpansionArea = []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	header.MessageCount = 234
	header.BodySize = 345

	return header
}

func createSmallTestAdminHeader() *ABlockHeader {
	header := new(ABlockHeader)

	p, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash, _ := NewShaHash(p)
	header.AdminChainID = hash
	p, _ = hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	hash, _ = NewShaHash(p)
	header.PrevFullHash = hash
	header.DBHeight = 123

	header.HeaderExpansionSize = 0
	header.HeaderExpansionArea = []byte{}
	header.MessageCount = 234
	header.BodySize = 345

	return header
}
