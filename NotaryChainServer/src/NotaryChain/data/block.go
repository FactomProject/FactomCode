package data

import (
	"errors"
	"crypto/sha256"
	"hash"
	"encoding/binary"
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
	b.Salt.writeToHash(sha)
	eh.writeToHash(sha)
	
	b.Salt = CreateHash(sha)
	
	return
}

func (b *Block) Hash() (hash *Hash, err error) {
	sha := sha256.New()
	
	if err = b.writeToHash(sha); err != nil {
		return
	}
	
	hash = CreateHash(sha)
	return
}

func (b *Block) writeToHash(sha hash.Hash) (err error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, b.BlockID)
	
	if _, err = sha.Write(buf); err != nil {
		return
	}
	
	if b.PreviousHash != nil {
		if err = b.PreviousHash.writeToHash(sha); err != nil {
			return
		}
	}
	
	for _,e := range b.Entries {
		if err = e.writeToHash(sha); err != nil {
			return
		}
	}
	
	err = b.Salt.writeToHash(sha)
	return
}