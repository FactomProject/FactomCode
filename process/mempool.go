// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package process

import (
	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/btcd/wire"
	"sync"
	"time"
)

// ftmMemPool is used as a source of factom transactions
// (CommitChain, RevealChain, CommitEntry, RevealEntry)
type ftmMemPool struct {
	sync.RWMutex
	pool        map[wire.ShaHash]wire.Message
	orphans     map[wire.ShaHash]wire.Message
	blockpool   map[string]wire.Message // to hold the blocks or entries downloaded from peers
	requested		map[common.Hash]*reqMsg
	lastUpdated time.Time               // last time pool was updated
}

type reqMsg struct {
	Msg *wire.MsgGetFactomData
	TimesMissed uint32
	Requested bool
}

// Add a factom message to the orphan pool
func (mp *ftmMemPool) init_ftmMemPool() error {

	mp.pool = make(map[wire.ShaHash]wire.Message)
	mp.orphans = make(map[wire.ShaHash]wire.Message)
	mp.blockpool = make(map[string]wire.Message)
	mp.requested = make(map[common.Hash]*reqMsg)

	return nil
}

func (mp *ftmMemPool) addMissingMsg(typ wire.InvType, hash *common.Hash, height uint32) *reqMsg {
	mp.Lock()
	defer mp.Unlock()

	h := common.NewHashFromByte(hash.ByteArray())
	if m, ok := mp.requested[h]; ok {
		m.TimesMissed++
		return m
	}
	inv := &wire.InvVectHeight {
		Type: typ,
		Hash: h,
		Height: height,
	}
	fd := wire.NewMsgGetFactomData()
	fd.AddInvVectHeight(inv)
	req := &reqMsg {
		Msg: fd,
		TimesMissed: 1,
	}
	mp.requested[h] = req
	return req
}

func (mp *ftmMemPool) removeMissingMsg(typ wire.InvType, hash *common.Hash) {
	mp.Lock()
	defer mp.Unlock()

	h := common.NewHashFromByte(hash.ByteArray())
	if _, ok := mp.requested[h]; ok {
		delete(mp.requested, h)
	}
}


// Add a factom message to the  Mem pool
func (mp *ftmMemPool) addMsg(msg wire.Message, hash *wire.ShaHash) error {
	mp.Lock()
	defer mp.Unlock()

	if len(mp.pool) > common.MAX_TX_POOL_SIZE {
		return errors.New("Transaction mem pool exceeds the limit.")
	}

	mp.pool[*hash] = msg

	return nil
}

// Add a factom message to the orphan pool
func (mp *ftmMemPool) addOrphanMsg(msg wire.Message, hash *wire.ShaHash) error {
	mp.Lock()
	defer mp.Unlock()

	if len(mp.orphans) > common.MAX_ORPHAN_SIZE {
		return errors.New("Ophan mem pool exceeds the limit.")
	}

	mp.orphans[*hash] = msg

	return nil
}

// Add a factom block message to the  Mem pool
func (mp *ftmMemPool) addBlockMsg(msg wire.Message, hash string) error {
	mp.Lock()
	defer mp.Unlock()

	if len(mp.blockpool) > common.MAX_BLK_POOL_SIZE {
		return errors.New("Block mem pool exceeds the limit. Please restart.")
	}
	mp.blockpool[hash] = msg

	return nil
}

// Delete a factom block message from the  Mem pool
func (mp *ftmMemPool) deleteBlockMsg(hash string) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.blockpool[hash] != nil {
		delete(fMemPool.blockpool, hash)
	}

	return nil
}
