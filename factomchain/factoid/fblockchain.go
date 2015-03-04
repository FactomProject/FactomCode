// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	//"bytes"
	//"fmt"
	"encoding/binary"
	"errors"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/btcd/wire"
	"sync"
	"time"
)

type FChain struct {
	ChainID      *notaryapi.Hash
	Name         [][]byte
	Blocks       []*FBlock
	CurrentBlock *FBlock
	BlockMutex   sync.Mutex
	NextBlockID  uint64
}

//Factoid Block header
type FBlockHeader struct {
	Height        uint64
	PrevBlockHash *notaryapi.Hash
	TimeStamp     int64
	TxCount       uint32
}

//Factoid Block - contains list of Tx, which has raw MsgTx plus the Txid
type FBlock struct {
	Header FBlockHeader
	//	Transactions []Tx

	//Not Marshalized
	FBHash   *notaryapi.Hash
	Salt     *notaryapi.Hash
	Chain    *FChain
	IsSealed bool
}

const (
	GenesisAddress = "FfZgRRHxuzsWkhXcb5Tb16EYuDEkbVCPAk1svfmYxyUXGPoS2X"
)

func FactoidGenesis(net wire.BitcoinNet) (genesis *FBlock) {
	genesis = &FBlock{
		Header: FBlockHeader{
			Height:    0,
			TimeStamp: 1424305819, //Thu, 19 Feb 2015 00:30:19 GMT
			TxCount:   1,
		},
	}

	return
}

func NewFBlockHeader(blockId uint64, prevHash *notaryapi.Hash, merkle *notaryapi.Hash) *FBlockHeader {
	return &FBlockHeader{
		PrevBlockHash: prevHash,
		TimeStamp:     time.Now().Unix(),
		Height:        blockId,
	}
}

func CreateFBlock(chain *FChain, prev *FBlock, cap uint) (b *FBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}

	b = new(FBlock)

	/*
		var prevHash *notaryapi.Hash
		if prev == nil {
			prevHash = notaryapi.NewHash()
		} else {
			prevHash, err = notaryapi.CreateHash(prev)
		}

		b.Header = *(NewFBlockHeader(chain.NextBlockID, prevHash, notaryapi.NewHash()))
		b.Chain = chain
		b.Transactions = make([]Tx, 0, cap)
		b.Salt = notaryapi.NewHash()
		b.IsSealed = false
	*/

	return b, err
}

func NewDBEntryFromFBlock(b *FBlock) *notaryapi.DBEntry {
	e := &notaryapi.DBEntry{}

	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(b.Header.TimeStamp))
	e.SetTimeStamp(bytes)

	e.ChainID = b.Chain.ChainID
	e.SetHash(b.FBHash.Bytes) // To be improved??
	e.MerkleRoot = b.FBHash   //To use MerkleRoot??

	return e
}
