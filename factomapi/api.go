// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package factomapi

import (
	"encoding/hex"

	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
)

var (
	db     database.Db
	inMsgQ chan wire.FtmInternalMsg
)

// TODO remove before production
func TestCredit(key []byte, amt int32) {
	msg := wire.NewMsgTestCredit()
	copy(msg.ECKey[:], key)
	msg.Amt = amt
	
	inMsgQ <- msg
}


func ChainHead(chainid string) (*common.EBlock, error) {
	h, err := atoh(chainid)
	if err != nil {
		return nil, err
	}
	c, err := db.FetchChainByHash(h)
	if err != nil {
		return nil, err
	}
	return c.NextBlock, nil
}

func CommitChain(c *common.CommitChain) error {
	m := wire.NewMsgCommitChain()
	m.CommitChain = c
	inMsgQ <- m
	return nil
}

func CommitEntry(c *common.CommitEntry) error {
	m := wire.NewMsgCommitEntry()
	m.CommitEntry = c
	inMsgQ <- m
	return nil
}

func DBlockByKeyMR(keymr string) (*common.DirectoryBlock, error) {
	h, err := atoh(keymr)
	if err != nil {
		return nil, err
	}
	r, err := db.FetchDBlockByHash(h)
	if err != nil {
		return r, err
	}
	return r, nil
}

func DBlockHead() (*common.DirectoryBlock, error) {
	bs, err := db.FetchAllDBlocks()
	if err != nil {
		return nil, err
	}
	return &bs[len(bs)-1], nil
}

func EBlockByKeyMR(keymr string) (*common.EBlock, error) {
	h, err := atoh(keymr)
	if err != nil {
		return nil, err
	}
	r, err := db.FetchEBlockByMR(h)
	if err != nil {
		return r, err
	}
	return r, nil
}

// TODO
func ECBalance(eckey string) (uint32, error) {
	return uint32(0), nil
}

func EntryByHash(hash string) (*common.Entry, error) {
	h, err := atoh(hash)
	if err != nil {
		return nil, err
	}
	r, err := db.FetchEntryByHash(h)
	if err != nil {
		return r, err
	}
	return r, nil
}

func RevealEntry(e *common.Entry) error {
	m := wire.NewMsgRevealEntry()
	m.Entry = e
	inMsgQ <- m
	return nil
}

func SetDB(d database.Db) {
	db = d
}

func SetInMsgQueue(q chan wire.FtmInternalMsg) {
	inMsgQ = q
}

func atoh(a string) (*common.Hash, error) {
	h := new(common.Hash)
	p, err := hex.DecodeString(a)
	if err != nil {
		return h, err
	}
	copy(h.Bytes(), p)
	return h, nil
}
