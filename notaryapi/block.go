package notaryapi

import (
	"bytes"
	"errors"
	
	"encoding/binary"
	"sync"
	"encoding/hex"
	
	//"github.com/firelizzard18/gocoding"
)

type Chain struct {
	ChainID 	*[]byte
	Blocks 		[]*Block
	BlockMutex 	sync.Mutex	
	NextBlockID uint64	
}




type Block struct {
	Chain *Chain
	
	BlockID uint64
	PreviousHash *Hash
	EBEntries []*EBEntry
	Salt *Hash
}
/*
type EntryBlock struct {
	BlockID uint64
	PreviousHash *Hash
	EBEntries []*EBEntry
	Salt *Hash
}
*/
/*func UpdateNextBlockID(id uint64) {
	nextBlockID = id
}
*/

func EncodeChainID(chainID *[]byte) (string){
	return hex.EncodeToString(*chainID)
}

func CreateBlock(chain *Chain, prev *Block, capacity uint) (b *Block, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}
	
	b = new(Block)
	
	b.BlockID = chain.NextBlockID
	b.Chain = chain
	chain.NextBlockID++
	
	b.EBEntries = make([]*EBEntry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	if prev == nil {
		b.PreviousHash = EmptyHash()
	} else {
		b.PreviousHash, err = CreateHash(prev)
	}
	
	return b, err
}




func (b *Block) AddEBEntry(e *Entry) (err error) {
	h, err := CreateHash(e)
	if err != nil { return }
	
	s, err := CreateHash(b.Salt, h)
	if err != nil { return }


 	ebEntry := NewEBEntry(h, b.Chain.ChainID)	
	
	b.EBEntries = append(b.EBEntries, ebEntry) 
	b.Salt = s
	
	return
}

func (b *Block) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PreviousHash.MarshalBinary()
	buf.Write(data)
	
	count := uint64(len(b.EBEntries))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i = i + 1 {
		data, _ := b.EBEntries[i].MarshalBinary()
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
	
	for _, ebentry := range b.EBEntries {
		size += ebentry.MarshalledSize()
	}
	
	return 0
}

func (b *Block) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.PreviousHash = new(Hash)
	b.PreviousHash.UnmarshalBinary(data)
	data = data[b.PreviousHash.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[0:8]), data[8:]
	b.EBEntries = make([]*EBEntry, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.EBEntries[i] = new(EBEntry)
		err = b.EBEntries[i].UnmarshalBinary(data)
		if err != nil { return }
		data = data[b.EBEntries[i].MarshalledSize():]
	}
	
	b.Salt = new(Hash)
	b.Salt.UnmarshalBinary(data)
	data = data[b.Salt.MarshalledSize():]
	
	return nil
}

/*func (b *Block) MarshallableFields() []gocoding.Field {
	return []gocoding.Field{
		gocoding.MakeField("blockID", b.BlockID, nil),
		gocoding.MakeField("previousHash", b.PreviousHash, nil),
		gocoding.MakeField("entries", b.Entries, nil),
		gocoding.MakeField("salt", b.Salt, nil), 
	}
}*/