package notarydata

import (
	"bytes"
	"errors"
	
	"encoding/binary"
)

var nextBlockID uint64 = 0

type Block struct {
	BlockID			uint64			`json:"blockID"`
	PreviousHash	*Hash			`json:"previousHash"`
	Entries			[]Entry			`json:"entries"`
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

func (b *Block) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 // BlockID uint64
	size += b.PreviousHash.MarshalledSize()
	size += 8 // len(Entries) uint64
	size += b.Salt.MarshalledSize()
	
	for _, entry := range b.Entries {
		size += entry.MarshalledSize()
	}
	
	return 0
}

func (b *Block) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:4]), data[4:]
	
	b.PreviousHash = new(Hash)
	b.PreviousHash.UnmarshalBinary(data)
	data = data[b.PreviousHash.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[0:4]), data[4:]
	b.Entries = make([]Entry, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.Entries[i], err = UnmarshalBinaryEntry(data)
		if err != nil { return }
		data = data[b.Entries[i].MarshalledSize():]
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
	
	b.Entries = make([]Entry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	if prev != nil {
		b.PreviousHash, err = CreateHash(prev)
	}
	
	return b, err
}

func (b *Block) AddEntry(e Entry) (err error) {
	h, err := CreateHash(e)
	if err != nil { return }
	
	s, err := CreateHash(b.Salt, h)
	if err != nil { return }
	
	b.Entries = append(b.Entries, e)
	b.Salt = s
	
	return
}