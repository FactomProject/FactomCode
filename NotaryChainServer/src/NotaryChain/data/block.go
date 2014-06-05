package data

import (
	"errors"
	"crypto/sha256"
	"encoding/binary"
	"bytes"
)

var nextBlockID uint64 = 0

type Block struct {
	BlockID			uint64			`json:"blockID"`
	PreviousHash	*Hash			`json:"previousHash"`
	Entries			[]*PlainEntry	`json:"entries"`
	Salt			*Hash			`json:"salt"`
}

func UpdateNextBlockID(id uint64) {
	nextBlockID = id
}

func (b *Block) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PreviousHash.MarshalBinary()
	buf.Write(data)
	
	count := uint64(len(b.Entries))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i = i + 1 {
		data, _ := b.Entries[i].MarshalBinary()
		buf.Write(data)
	}
	
	data, _ = b.Salt.MarshalBinary()
	buf.Write(data)
	
	return buf.Bytes(), err
}

func (b *Block) UnmarshalBinary(data []byte) error {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:4]), data[4:]
	
	b.PreviousHash = new(Hash)
	b.PreviousHash.UnmarshalBinary(data)
	data = data[b.PreviousHash.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[0:4]), data[4:]
	b.Entries = make([]*PlainEntry, 0, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.Entries[0] = new(PlainEntry)
		b.Entries[0].UnmarshalBinary(data)
		data = data[b.Entries[0].MarshalledSize():]
	}
	
	b.Salt = new(Hash)
	b.Salt.UnmarshalBinary(data)
	data = data[b.Salt.MarshalledSize():]
	
	return nil
}

func CreateBlock(prev *Block, capacity uint) (b *Block, err error) {
	if prev == nil && nextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && nextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}
	
	b = new(Block)
	
	b.BlockID = nextBlockID
	nextBlockID++
	
	b.Entries = make([]*PlainEntry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	if prev != nil {
		b.PreviousHash, err = prev.Hash()
	}
	
	return b, err
}

func (b *Block) AddEntry(e *PlainEntry) (err error) {
	b.Entries = append(b.Entries, e)
	
	var eh *Hash
	eh, err = e.Hash();
	if err != nil { return err }
	
	sha := sha256.New()
	
	data, _ := b.Salt.MarshalBinary()
	sha.Write(data)
	
	data, _ = eh.MarshalBinary()
	sha.Write(data)
	
	b.Salt = CreateHash(sha)
	
	return
}

func (b *Block) Hash() (hash *Hash, err error) {
	sha := sha256.New()
	
	data, _ := b.MarshalBinary()
	sha.Write(data)
	
	hash = CreateHash(sha)
	return
}