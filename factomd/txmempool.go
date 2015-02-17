// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package main 

import (
	//"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
)


// TxProcessor is an interface that abstracts methods needed to process a TxMessage 
// SetContext(*TxMessage) will store to concrete TxMessage in the concrete TxProcessor object 
type TxProcessor interface {
	SetContext(*factomwire.MsgTx)
	Verify() bool
	//Broadcast() 
	AddToMemPool() 
}

//ProcessTx is an abstract way to process any TxMessage received from wire 
//It first SetContext()  of the concrete type and then will only Broadcast and 
//add to MemoryyPool if Verify returns true
func ProcessTx(tp TxProcessor, tm *factomwire.MsgTx) (ret bool) {
	tp.SetContext(tm)
	if tp.Verify() {
		//tp.server.BroadcastMessage()
		tp.AddToMemPool()
		ret = true
	}

	return
}

type txMemPool struct {
	server	*server
	TxProcessorS map[string]TxProcessor
}

func (txm *txMemPool) ProcessTransaction(tm *factomwire.MsgTx) bool { //factomwire.MsgTx) {
	return ProcessTx(txm.TxProcessorS[tm.TxType()],tm)
}

// newTxMemPool returns a new memory pool for validating and storing standalone
// transactions until they are confirmed into a block.
func newTxMemPool(server *server) *txMemPool {
	var txmp txMemPool
	txmp.TxProcessorS = make(map[string]TxProcessor)

	txmp.TxProcessorS["factoid"] = factoid.NewFactoidPool()

	//txmp.TxProcessorS["entrycommit"] = factom.NewEntryPool()

	return &txmp
}
