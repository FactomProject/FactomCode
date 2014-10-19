package notaryapi

import (
	"bytes"
	"time"
	
	"encoding/binary"
)

const FBlockVersion = 1 

type FBlockHeader struct {
	BlockID uint64
	PrevBlockHash *Hash
	MerkleRoot *Hash
	Version int32
	TimeStamp int64
	EntryCount uint32
}

const fBlockHeaderLen = 88

func (b *FBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PrevBlockHash.MarshalBinary()
	buf.Write(data)
	
	data, _ = b.MerkleRoot.MarshalBinary()
	buf.Write(data)
		
	binary.Write(&buf, binary.BigEndian, b.Version)
	binary.Write(&buf, binary.BigEndian, b.TimeStamp)
	binary.Write(&buf, binary.BigEndian, b.EntryCount)
	
	return buf.Bytes(), err
}

func (b *FBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 
	size += b.PrevBlockHash.MarshalledSize()
	size += b.MerkleRoot.MarshalledSize()
	size += 4
	size += 8
	size += 4
	
	return size
}

func (b *FBlockHeader) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.PrevBlockHash = new(Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]
	
	b.MerkleRoot = new(Hash)
	b.MerkleRoot.UnmarshalBinary(data)
	data = data[b.MerkleRoot.MarshalledSize():]
	
	version, data := binary.BigEndian.Uint32(data[0:4]), data[4:]
	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
	b.EntryCount, data = binary.BigEndian.Uint32(data[0:4]), data[4:]
	
	b.Version = int32(version)
	b.TimeStamp = int64(timeStamp)

	return nil
}


func NewFBlockHeader(blockId uint64, prevHash *Hash, merkleRootHash *Hash, 
	version int32, count uint32) *FBlockHeader {

	return &FBlockHeader{
		Version:    version,
		PrevBlockHash:  prevHash,
		MerkleRoot: merkleRootHash,
		TimeStamp:  time.Now().Unix(),
		EntryCount: count,
		BlockID:    blockId,
	}
}

func (b *FBlockHeader) RealTime() time.Time {
	return time.Unix(b.TimeStamp, 0)
}