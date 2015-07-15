package common_test

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "testing"

    . "github.com/FactomProject/FactomCode/common"
)

func TestMarshalUnmarshalDirectoryBlockHeader(t *testing.T) {
    fmt.Println("\n---\nTestMarshalUnmarshalDirectoryBlockHeader\n---\n")

    header := createTestDirectoryBlockHeader()

    bytes1, err := header.MarshalBinary()
    t.Logf("bytes1: %X\n", bytes1)

    header2 := new(DBlockHeader)
    header2.UnmarshalBinary(bytes1)

    bytes2, err := header2.MarshalBinary()
    if err != nil {
        t.Errorf("Error:%v", err)
    }
    t.Logf("bytes2: %X\n", bytes2)

    if bytes.Compare(bytes1, bytes2) != 0 {
        t.Errorf("Invalid output")
    }

}

func TestMarshalUnmarshalDirectoryBlock(t *testing.T) {
    fmt.Println("\n---\nTestMarshalUnmarshalDirectoryBlock\n---\n")

    dblock := createTestDirectoryBlock()

    bytes1, err := dblock.MarshalBinary()
    t.Logf("bytes1: %X\n", bytes1)

    dblock2 := new(DirectoryBlock)
    dblock2.UnmarshalBinary(bytes1)

    bytes2, err := dblock2.MarshalBinary()
    if err != nil {
        t.Errorf("Error: %v", err)
    }
    t.Logf("bytes2: %X\n", bytes2)

    if bytes.Compare(bytes1, bytes2) != 0 {
        t.Errorf("Invalid output")
    }
}

func TestMakeSureBlockCountIsNotDuplicates(t *testing.T) {
    fmt.Println("\n---\nTestMakeSureBlockCountIsNotDuplicates\n---\n")
    block := createTestDirectoryBlock()
    block.DBEntries = []*DBEntry{}
    min := 1000
    max := -1

    for i := 0; i < 100; i++ {
        //Update the BlockCount in header
        block.Header.BlockCount = uint32(len(block.DBEntries))
        //Marshal the block
        marshalled, err := block.MarshalBinary()
        if err != nil {
            t.Errorf("Error: %v", err)
        }
        //Get the byte representation of BlockCount
        var buf bytes.Buffer
        binary.Write(&buf, binary.BigEndian, block.Header.BlockCount)
        hex := buf.Bytes()

        //How many times does BlockCount appear in the marshalled slice?
        count := bytes.Count(marshalled, hex)
        if count > max {
            max = count
        }
        if count < min {
            min = count
        }

        de := new(DBEntry)
        de.ChainID = NewHash()
        de.KeyMR = NewHash()

        block.DBEntries = append(block.DBEntries, de)
    }
    t.Logf("Min count - %v, max count - %v", min, max)
    if min != 1 {
        t.Errorf("Invalid number of BlockCount occurances")
    }
}

func createTestDirectoryBlock() *DirectoryBlock {
    dblock := new(DirectoryBlock)

    dblock.Header = createTestDirectoryBlockHeader()

    dblock.DBEntries = make([]*DBEntry, 0, 5)

    de := new(DBEntry)
    de.ChainID = NewHash()
    de.KeyMR = NewHash()

    dblock.DBEntries = append(dblock.DBEntries, de)
    dblock.Header.BlockCount = uint32(len(dblock.DBEntries))

    return dblock
}

func createTestDirectoryBlockHeader() *DBlockHeader {
    header := new(DBlockHeader)

    header.DBHeight = 1
    header.BodyMR = NewHash()
    header.BlockCount = 0
    header.NetworkID = 9
    header.PrevFullHash = NewHash()
    header.PrevKeyMR = NewHash()
    header.Timestamp = 1234
    header.Version = 1

    return header
}
