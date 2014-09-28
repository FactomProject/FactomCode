package notaryapi

import (
	"bytes"
	"errors"
	
	"encoding/binary"
	"sync"
)

type FChain struct {
	ChainID 	*[]byte
	Blocks 		[]*FBlock
	BlockMutex 	sync.Mutex	
	NextBlockID uint64	
}

type FBlock struct {
	Chain *FChain
	
	BlockID uint64
	PreviousHash *Hash
	FBEntries []*FBEntry
	Salt *Hash
	
	Sealed bool //?
}


func CreateFBlock(chain *FChain, prev *FBlock, capacity uint) (b *FBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}
	
	b = new(FBlock)
	
	b.BlockID = chain.NextBlockID
	b.Chain = chain
	chain.NextBlockID++
	
	b.FBEntries = make([]*FBEntry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	if prev == nil {
		b.PreviousHash = EmptyHash()
	} else {
		b.PreviousHash, err = CreateHash(prev)
	}
	
	return b, err
}

// Add FBEntry from an Entry Block
func (fchain *FChain) AddFBEntry(eb *Block) (err error) {
	fBlock := fchain.Blocks[len(fchain.Blocks)-1]
	hash, _ := CreateHash (eb) // redundent work??
	
	fbEntry := NewFBEntry(hash, eb.Chain.ChainID)
	fBlock.AddFBEntry(fbEntry)
	
	return nil
	
	}


func (b *FBlock) AddFBEntry(e *FBEntry) (err error) {
	h, err := CreateHash(e)
	if err != nil { return }
	
	s, err := CreateHash(b.Salt, h)
	if err != nil { return }


 	fbEntry := NewFBEntry(h, b.Chain.ChainID)	
	
	b.FBEntries = append(b.FBEntries, fbEntry) 
	b.Salt = s
	
	return
}

func (b *FBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PreviousHash.MarshalBinary()
	buf.Write(data)
	
	if b.Sealed == true{
		count := uint64(len(b.FBEntries))
		binary.Write(&buf, binary.BigEndian, count)
		for i := uint64(0); i < count; i = i + 1 {
			data, _ := b.FBEntries[i].MarshalBinary()
			buf.Write(data)
		}
	} else{
		binary.Write(&buf, binary.BigEndian, uint64(0))
	}
	
	data, _ = b.Salt.MarshalBinary()
	buf.Write(data)
	
	return buf.Bytes(), err
}

func (b *FBlock) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 // BlockID uint64
	size += b.PreviousHash.MarshalledSize()
	size += 8 // len(Entries) uint64
	size += b.Salt.MarshalledSize()
	
	for _, fbentry := range b.FBEntries {
		size += fbentry.MarshalledSize()
	}
	
	return 0
}

func (b *FBlock) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.PreviousHash = new(Hash)
	b.PreviousHash.UnmarshalBinary(data)
	data = data[b.PreviousHash.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[0:8]), data[8:]
	b.FBEntries = make([]*FBEntry, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.FBEntries[i] = new(FBEntry)
		err = b.FBEntries[i].UnmarshalBinary(data)
		if err != nil { return }
		data = data[b.FBEntries[i].MarshalledSize():]
	}
	
	b.Salt = new(Hash)
	b.Salt.UnmarshalBinary(data)
	data = data[b.Salt.MarshalledSize():]
	
	return nil
}
