package notaryapi

import (
	"bytes"
	"time"
	
	"encoding/binary"
)


type EBlockHeader struct {
//	ChainID []byte // ?? put in ebinfo for now
	BlockID uint64
	PrevBlockHash *Hash
	//MerkleRoot *Hash
	TimeStamp int64
}

const eBlockHeaderLen = 80

func (b *EBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PrevBlockHash.MarshalBinary()
	buf.Write(data)
/*	
	data, _ = b.MerkleRoot.MarshalBinary()
	buf.Write(data)
*/		
	binary.Write(&buf, binary.BigEndian, b.TimeStamp)
	
	return buf.Bytes(), err
}

func (b *EBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 
	size += b.PrevBlockHash.MarshalledSize()
//	size += b.MerkleRoot.MarshalledSize()
	size += 8
	
	return size
}

func (b *EBlockHeader) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.PrevBlockHash = new(Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]
/*	
	b.MerkleRoot = new(Hash)
	b.MerkleRoot.UnmarshalBinary(data)
	data = data[b.MerkleRoot.MarshalledSize():]
*/	
	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
	b.TimeStamp = int64(timeStamp)

	return nil
}


func NewEBlockHeader(blockId uint64, prevHash *Hash, merkle *Hash) *EBlockHeader {

	return &EBlockHeader{
		PrevBlockHash:  prevHash,
		//MerkleRoot: merkle,
		TimeStamp:  time.Now().Unix(),
		BlockID:    blockId,
	}
}

func (e *EBlockHeader) RealTime() time.Time {
	return time.Unix(e.TimeStamp, 0)
}
