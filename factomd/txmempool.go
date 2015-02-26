// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package main

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
)

// TxProcessor is an interface that abstracts methods needed to process a TxMessage
// SetContext(*TxMessage) will store to concrete TxMessage in the concrete TxProcessor object
type TxProcessor interface {
	SetContext(*factomwire.MsgTx)
	Verify() bool
	//Broadcast()
	AddToMemPool() []*notaryapi.HashF
	Confirm(*factomwire.MsgConfirmation) (bool, []*notaryapi.HashF)
}

//ProcessTx is an abstract way to process any TxMessage received from wire
//It first SetContext()  of the concrete type and then will only Broadcast and
//add to MemoryyPool if Verify returns true
func ProcessTx(tp TxProcessor, tm *factomwire.MsgTx) (ret bool, verified []*notaryapi.HashF) {
	tp.SetContext(tm)
	if ret = tp.Verify(); ret {
		//tp.server.BroadcastMessage()
		verified = tp.AddToMemPool()
	}

	return
}

func ProcessConf(tp TxProcessor, tm *factomwire.MsgConfirmation) (ret bool, confirmed []*notaryapi.HashF) {
	ret, confirmed = tp.Confirm(tm)
	return
}

type txMemPool struct {
	server       *server
	TxProcessorS map[string]TxProcessor
	//have tx need conf
	waitingconf map[notaryapi.HashF]*factomwire.MsgTx
	//have conf need tx
	waitingtx  map[notaryapi.HashF]*factomwire.MsgConfirmation
	orphanconf map[uint32]*factomwire.MsgConfirmation
	next       uint32
}

func (txm *txMemPool) ProcessTransaction(tm *factomwire.MsgTx) bool {
	ok, verified := ProcessTx(txm.TxProcessorS[tm.TxType()], tm)
	if !ok {
		return false
	}

	for _, v := range verified {
		if conf, ok := txm.waitingtx[*v]; ok {
			txm.waitingconf[*v] = tm
			//txm.ProcessConfirmation(conf)
			delete(txm.waitingtx, *v)
			defer func() { inRpcQueue <- conf }()
		} else {
			txm.waitingconf[*v] = tm
			if y := ImResponsible(v); y {
				defer func() {
					inRpcQueue <- txm.server.fedprocesslist.Confirm(v)
				}()
			}
		}
	}

	return true
}

func (txm *txMemPool) ProcessConfirmation(tm *factomwire.MsgConfirmation) (ok bool) {
	tx, k := txm.waitingconf[tm.Affirmation]
	if !k {
		txm.waitingtx[tm.Affirmation] = tm
		return k
	}

	if ok = VerifyResponsible((*notaryapi.HashF)(&tm.Affirmation), tm.Affirmation[:], &tm.Signature); !ok {
		return
	}

	if tm.Index > txm.next {
		txm.orphanconf[tm.Index] = tm
		return true
	} else if tm.Index < txm.next {
		return false
	}

	ok = txm.doConf(tx, tm)

	for {
		txm.next++
		if nc, y := txm.orphanconf[txm.next]; y {
			tx, k := txm.waitingconf[nc.Affirmation]
			if !k {
				panic("should have tx waitingconf")
			}
			txm.doConf(tx, nc)
			delete(txm.orphanconf, txm.next)
		} else {
			break
		}
	}

	return
}

func (txm *txMemPool) doConf(itx *factomwire.MsgTx, tm *factomwire.MsgConfirmation) bool {
	ret, confirmed := ProcessConf(txm.TxProcessorS[itx.TxType()], tm)
	for _, c := range confirmed {
		if tx, ok := txm.waitingconf[*c]; ok {
			//kludge to write blocks. processed by restapi for now
			//ToDo: fix
			defer func() { inMsgQueue <- tx }()
			delete(txm.waitingconf, *c)
		}
	}

	return ret
}

// newTxMemPool returns a new memory pool for validating and storing standalone
// transactions until they are confirmed into a block.
func newTxMemPool(server *server) *txMemPool {
	var txmp txMemPool
	txmp.next = 1
	txmp.TxProcessorS = make(map[string]TxProcessor)

	txmp.TxProcessorS["factoid"] = factoid.NewFactoidPool()

	//txmp.TxProcessorS["entrycommit"] = factom.NewEntryPool()
	
	// Please double check ?? --------------------------------------
	//have tx need conf
	txmp.waitingconf  = make (map[notaryapi.HashF]*factomwire.MsgTx)
	//have conf need tx
	txmp.waitingtx  = make (map[notaryapi.HashF]*factomwire.MsgConfirmation)
	txmp.orphanconf = make (map[uint32]*factomwire.MsgConfirmation)
	//----------------------------------------------------------------

	return &txmp
}
