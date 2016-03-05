// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	"github.com/FactomProject/FactomCode/wire"
)

// ftmMemPool is used as a source of factom transactions
// (CommitChain, RevealChain, CommitEntry, RevealEntry, Ack, EOM, DirBlockSig)
type ftmMemPool struct {
	sync.RWMutex
	pool             map[wire.ShaHash]wire.Message
	orphans          map[wire.ShaHash]wire.Message
	blockpool        map[string]wire.Message // to hold the blocks or entries downloaded from peers
	ackpool          []*wire.MsgAck
	dirBlockSigs     []*wire.MsgDirBlockSig
	processListItems []*consensus.ProcessListItem
	lastUpdated      time.Time // last time pool was updated
}

// Add a factom message to the orphan pool
func (mp *ftmMemPool) initFtmMemPool() error {
	mp.pool = make(map[wire.ShaHash]wire.Message)
	mp.orphans = make(map[wire.ShaHash]wire.Message)
	mp.blockpool = make(map[string]wire.Message)
	mp.ackpool = make([]*wire.MsgAck, 100, 20000)
	mp.dirBlockSigs = make([]*wire.MsgDirBlockSig, 0, 32)
	mp.processListItems = make([]*consensus.ProcessListItem, 100, 20000)
	return nil
}

func (mp *ftmMemPool) addDirBlockSig(dbsig *wire.MsgDirBlockSig) {
	mp.dirBlockSigs = append(mp.dirBlockSigs, dbsig)
}

func (mp *ftmMemPool) LenDirBlockSig() int {
	return len(mp.dirBlockSigs)
}

func (mp *ftmMemPool) resetDirBlockSigPool() {
	fmt.Println("resetDirBlockSigPool")
	mp.dirBlockSigs = make([]*wire.MsgDirBlockSig, 0, 32)
}

func (mp *ftmMemPool) getDirBlockSigPool() []*wire.MsgDirBlockSig {
	return mp.dirBlockSigs
}

// addAck add the ack to ackpool and find it's acknowledged msg.
// then add them to ftmMemPool if available.
// otherwise return missing acked msg.
func (mp *ftmMemPool) addAck(ack *wire.MsgAck) *wire.MsgMissing {
	if ack == nil {
		return nil
	}
	procLog.Infof("ftmMemPool.addAck: %+v\n", ack)
	// check duplication first
	a := mp.ackpool[ack.Index]
	if a != nil && ack.Equals(a) {
		fmt.Println("duplicated ack: ignore it")
		return nil
	}
	mp.ackpool[ack.Index] = ack
	if ack.Type == wire.AckRevealEntry || ack.Type == wire.AckRevealChain ||
		ack.Type == wire.AckCommitChain || ack.Type == wire.AckCommitEntry {
		msg := mp.pool[*ack.Affirmation]
		if msg == nil {
			// missing msg and request it from the leader ???
			// create a new msg type
			return wire.NewMsgMissing(ack.Height, ack.Index, ack.Type)
		} else {
			pli := &consensus.ProcessListItem{
				Ack:     ack,
				Msg:     msg,
				MsgHash: ack.Affirmation,
			}
			mp.processListItems[ack.Index] = pli
		}
	} else if ack.IsEomAck() {
		pli := &consensus.ProcessListItem{
			Ack:     ack,
			Msg:     ack,
			MsgHash: nil,
		}
		mp.processListItems[ack.Index] = pli
	}
	return nil
}

func (mp *ftmMemPool) getMissingMsgAck(ack *wire.MsgAck) []*wire.MsgMissing {
	var i uint32
	var missingAcks []*wire.MsgMissing
	for i = 0; i <= ack.Index; i++ {
		if mp.ackpool[i] == nil {
			// missing an ACK here. request for this ack
			m := wire.NewMsgMissing(ack.Height, ack.Index, ack.Type)
			missingAcks = append(missingAcks, m)
			procLog.Infof("Missing an Ack at index=%d", i)
		}
	}
	return missingAcks
}

func (mp *ftmMemPool) assembleFollowerProcessList(ack *wire.MsgAck) error {
	//do a simple happy path for now ???
	var height uint32
	for i := len(mp.ackpool) - 1; i >= 0; i-- {
		if mp.ackpool[i] != nil {
			height = mp.ackpool[i].Height
			break
		}
	}
	plMgr.NextDBlockHeight = height
	fmt.Println("assembleFollowerProcessList: (plMgr.NextDBlockHeight) target height=", height)
	for i := 0; i < 10; i++ {
		msgEom := &wire.MsgInt_EOM{
			EOM_Type:         wire.EndMinute1 + byte(i),
			NextDBlockHeight: height,
		}
		plMgr.AddToLeadersProcessList(msgEom, nil, msgEom.EOM_Type)
		if ack.ChainID == nil {
			ack.ChainID = dchain.ChainID
		}
	}
	return nil

	//???
	/*
		// simply validation
		if ack.Type != wire.EndMinute10 || mp.ackpool[len(mp.ackpool)-1] != ack {
			return fmt.Errorf("the last ack has to be EndMinute10")
		}
		procLog.Info("acpool.len=", len(mp.ackpool)) //, spew.Sdump(mp.ackpool))
		for i := 0; i < len(mp.ackpool); i++ {
			if mp.ackpool[i] == nil {
				// missing an ACK here
				// todo: request for this ack, panic for now
				//panic(fmt.Sprintf("Missing an Ack in ackpool at index=%d, for ack=%s", i, spew.Sdump(ack)))
				//procLog.Infof("Missing an Ack in ackpool at index=%d\n", i)
				continue
			}
			var msg wire.Message
			if mp.ackpool[i].Affirmation != nil {
				msg = mp.pool[*mp.ackpool[i].Affirmation]
			}
			plMgr.AddToFollowersProcessList(msg, mp.ackpool[i])
		}
		return nil
	*/
}

// Add a factom message to the  Mem pool
func (mp *ftmMemPool) addMsg(msg wire.Message, hash *wire.ShaHash) error {
	if len(mp.pool) > common.MAX_TX_POOL_SIZE {
		return errors.New("Transaction mem pool exceeds the limit.")
	}
	mp.pool[*hash] = msg
	return nil
}

// Add a factom message to the orphan pool
func (mp *ftmMemPool) addOrphanMsg(msg wire.Message, hash *wire.ShaHash) error {
	if len(mp.orphans) > common.MAX_ORPHAN_SIZE {
		return errors.New("Ophan mem pool exceeds the limit.")
	}
	mp.orphans[*hash] = msg
	return nil
}

// Add a factom block message to the  Mem pool
func (mp *ftmMemPool) addBlockMsg(msg wire.Message, hash string) error {
	if len(mp.blockpool) > common.MAX_BLK_POOL_SIZE {
		return errors.New("Block mem pool exceeds the limit. Please restart.")
	}
	mp.Lock()
	mp.blockpool[hash] = msg
	mp.Unlock()
	return nil
}

// Delete a factom block message from the  Mem pool
func (mp *ftmMemPool) deleteBlockMsg(hash string) error {
	if mp.blockpool[hash] != nil {
		mp.Lock()
		delete(fMemPool.blockpool, hash)
		mp.Unlock()
	}
	return nil
}
