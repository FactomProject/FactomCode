package data

import (
	"errors"
)

var nextBlockID int64 = 0

type Block struct {
	BlockID			int64
	PreviousHash	*Hash
	Entries			[]Entry
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
	
	if prev == nil {
		return b, nil
	}
	
	b.PreviousHash, err = prev.Hash()
	
	return b, err
}

func (b *Block) Hash() (hash *Hash, err error) {
	return new(Hash), nil
}