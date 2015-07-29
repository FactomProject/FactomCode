package common_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)

func TestAdminBlockPreviousHash(t *testing.T) {
	fmt.Printf("\n---\nTestAdminBlockMarshalUnmarshal\n---\n")

	block := new(AdminBlock)
	data, _ := hex.DecodeString("000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	_, err := block.UnmarshalBinaryData(data)
	if err != nil {
		t.Error(err)
	}

	fullHash, err := block.FullHash()
	if err != nil {
		t.Error(err)
	}

	partialHash, err := block.PartialHash()
	if err != nil {
		t.Error(err)
	}

	t.Logf("Current hashes - %s, %s", fullHash.String(), partialHash.String())

	if fullHash.String() != "0a9aa1efbe7d0e8d9c1d460d1c78e3e7b50f984e65a3f3ee7b73100a94189dbf" {
		t.Error("Invalid fullHash")
	}
	if partialHash.String() != "4fb409d5369fad6aa7768dc620f11cd219f9b885956b631ad050962ca934052e" {
		t.Error("Invalid partialHash")
	}

	aChain := new(AdminChain)
	aChain.NextBlockHeight = 1
	aChain.ChainID = block.Header.AdminChainID

	block2, err := CreateAdminBlock(aChain, block, 5)
	if err != nil {
		t.Error(err)
	}

	fullHash2, err := block2.FullHash()
	if err != nil {
		t.Error(err)
	}

	partialHash2, err := block2.PartialHash()
	if err != nil {
		t.Error(err)
	}

	t.Logf("Second hashes - %s, %s", fullHash2.String(), partialHash2.String())
	t.Logf("Previous hash - %s", block2.Header.PrevFullHash.String())

	marshalled, err := block2.MarshalBinary()
	if err != nil {
		t.Error(err)
	}
	t.Logf("Marshalled - %X", marshalled)

	if block2.Header.PrevFullHash.String() != fullHash.String() {
		t.Error("PrevFullHash does not match ABHash")
	}
}

func TestAdminBlockMarshalUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestAdminBlockMarshalUnmarshal\n---\n")

	blocks := []*AdminBlock{}
	blocks = append(blocks, createSmallTestAdminBlock())
	blocks = append(blocks, createTestAdminBlock())
	for b, block := range blocks {
		binary, err := block.MarshalBinary()
		if err != nil {
			t.Logf("Block %d", b)
			t.Error(err)
			t.FailNow()
		}
		block2 := new(AdminBlock)
		err = block2.UnmarshalBinary(binary)
		if err != nil {
			t.Logf("Block %d", b)
			t.Error(err)
			t.FailNow()
		}
		if len(block2.ABEntries) != len(block.ABEntries) {
			t.Logf("Block %d", b)
			t.Error("Invalid amount of ABEntries")
			t.FailNow()
		}
		for i := range block2.ABEntries {
			entryOne, err := block.ABEntries[i].MarshalBinary()
			if err != nil {
				t.Logf("Block %d", b)
				t.Error(err)
				t.FailNow()
			}
			entryTwo, err := block2.ABEntries[i].MarshalBinary()
			if err != nil {
				t.Logf("Block %d", b)
				t.Error(err)
				t.FailNow()
			}

			if bytes.Compare(entryOne, entryTwo) != 0 {
				t.Logf("Block %d", b)
				t.Logf("%X vs %X", entryOne, entryTwo)
				t.Error("ABEntries are not identical")
			}
		}
	}
}

func TestABlockHeaderMarshalUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestABlockHeaderMarshalUnmarshal\n---\n")

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

var WeDidPanic bool

func CatchPanic() {
	if r := recover(); r != nil {
		WeDidPanic = true
	}
}

func TestInvalidABlockHeaderUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestInvalidABlockHeaderUnmarshal\n---\n")

	WeDidPanic = false
	defer CatchPanic()

	header := new(ABlockHeader)
	err := header.UnmarshalBinary(nil)
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	if WeDidPanic == true {
		t.Error("We did panic and we shouldn't have")
		WeDidPanic = false
		defer CatchPanic()
	}

	header = new(ABlockHeader)
	err = header.UnmarshalBinary([]byte{})
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	if WeDidPanic == true {
		t.Error("We did panic and we shouldn't have")
		WeDidPanic = false
		defer CatchPanic()
	}

	header2 := createTestAdminHeader()

	binary, err := header2.MarshalBinary()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	header = new(ABlockHeader)
	err = header.UnmarshalBinary(binary[:len(binary)-1])
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	if WeDidPanic == true {
		t.Error("We did panic and we shouldn't have")
		WeDidPanic = false
		defer CatchPanic()
	}
}

func TestInvalidAdminBlockUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestInvalidAdminBlockUnmarshal\n---\n")

	WeDidPanic = false
	defer CatchPanic()

	block := new(AdminBlock)
	err := block.UnmarshalBinary(nil)
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	if WeDidPanic == true {
		t.Error("We did panic and we shouldn't have")
		WeDidPanic = false
		defer CatchPanic()
	}

	block = new(AdminBlock)
	err = block.UnmarshalBinary([]byte{})
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	if WeDidPanic == true {
		t.Error("We did panic and we shouldn't have")
		WeDidPanic = false
		defer CatchPanic()
	}

	block2 := createTestAdminBlock()

	binary, err := block2.MarshalBinary()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	block = new(AdminBlock)
	err = block.UnmarshalBinary(binary[:len(binary)-1])
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	if WeDidPanic == true {
		t.Error("We did panic and we shouldn't have")
		WeDidPanic = false
		defer CatchPanic()
	}
}

func TestMarshalledSize(t *testing.T) {
	fmt.Printf("\n---\nTestMarshalledSize\n---\n")

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

func createTestAdminBlock() *AdminBlock {
	block := new(AdminBlock)
	block.Header = createTestAdminHeader()

	p, _ := hex.DecodeString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	hash, _ := NewShaHash(p)
	sigBytes := make([]byte, 96)
	for i := 0; i < 5; i++ {
		for j := range sigBytes {
			sigBytes[j] = byte(i)
		}
		sig := UnmarshalBinarySignature(sigBytes)
		entry := NewDBSignatureEntry(hash, sig)
		block.ABEntries = append(block.ABEntries, entry)
	}

	block.Header.MessageCount = uint32(len(block.ABEntries))
	return block
}

func createSmallTestAdminBlock() *AdminBlock {
	block := new(AdminBlock)
	block.Header = createSmallTestAdminHeader()
	block.Header.MessageCount = uint32(len(block.ABEntries))
	return block
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
