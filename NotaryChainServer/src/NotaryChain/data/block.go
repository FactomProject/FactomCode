package data

import (
	"errors"
	"crypto/sha256"
	"hash"
	"encoding/binary"
)

var nextBlockID uint64 = 0

type Block struct {
	BlockID			uint64
	PreviousHash	*Hash
	Entries			[]*Entry
	Salt			*Hash
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
	
	b.Entries = make([]*Entry, 0, capacity)
	
	b.Salt = new(Hash)
	
	if prev == nil {
		return b, nil
	}
	
	b.PreviousHash, err = prev.Hash()
	
	return b, err
}

func (b *Block) AddEntry(e *Entry) (err error) {
	b.Entries = append(b.Entries, e)
	
	var eh *Hash
	eh, err = e.Hash();
	if err != nil { return err }
	
	h := sha256.New()
	b.Salt.writeToHash(h)
	eh.writeToHash(h)
	
	b.Salt = CreateHash(h)
	
	return nil;
}

func (b *Block) Hash() (hash *Hash, err error) {
	h := sha256.New()
	
	if err = b.writeToHash(h); err != nil {
		return nil, err
	}
	
	return CreateHash(h), nil
}

func (b *Block) writeToHash(h hash.Hash) (err error) {
	var buf []byte
	binary.BigEndian.PutUint64(buf, b.BlockID)
	
	if _, err = h.Write(buf); err != nil {
		return err
	}
	
	if err = b.PreviousHash.writeToHash(h); err != nil {
		return err
	}
	
	for _,e := range b.Entries {
		if err = e.writeToHash(h); err != nil {
			return err
		}
	}
	
	err = b.Salt.writeToHash(h)
	return err
}