// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"github.com/FactomProject/FactomCode/factomwire"
	"sync"
)

const (
	//initail array allocation
	//each call to append will double capacity when len() = cap()
	TxPoolAllocSize = 1000
)

// ToDo: TxProcessor was copied from factomd.txmempool due to import cycle issues. fix this
//
// TxProcessor is an interface that abstracts methods needed to process a TxMessage
// SetContext(*TxMessage) will store to concrete TxMessage in the concrete TxProcessor object
type TxProcessor interface {
	SetContext(*factomwire.MsgTx)
	Verify() bool
	//Broadcast()
	AddToMemPool()
}

// txMemPool is used as a source of transactions that need to be mined into
// blocks and relayed to other peers.  It is safe for concurrent access from
// multiple peers.
//   factoidpool impliments TxProcessor for generic mempool processing
type factoidPool struct {
	TxProcessor
	sync.RWMutex
	//server        *server
	utxo    Utxo
	context context
	txpool  *txpool
}

//the context is the tx used by default when no tx is referenced
// this is needed in order for mempool methods to be called from abstract interface
// when different mempools have different tx structs  (no plyorphism in go)
type context struct {
	wire  *factomwire.MsgTx
	tx    *Tx
	index int //index into txpool txindex array of *Txid
}

//txpool stores an array of Txids in order of insertion and
//a map from Txid to the context tx
type txpool struct {
	txindex []*Txid
	txlist  map[Txid]context
	//nexti		int
}

//create new txpool, allocate TxPoolAllocSize
func NewTxPool() *txpool {
	return &txpool{
		txindex: make([]*Txid, 0, TxPoolAllocSize),
		txlist:  make(map[Txid]context),
		//nexti: 		0
	}
}

// newTxMemPool returns a new memory pool for validating and storing standalone
// transactions until they are mined into a block.
func NewFactoidPool() *factoidPool {
	return &factoidPool{
		//server:        server,
		utxo:   NewUtxo(),
		txpool: NewTxPool(),
	}
}

//add context tx to txpool
func (tp *txpool) AddContext(c *context) {
	c.index = len(tp.txindex)
	tp.txindex = append(tp.txindex, c.tx.Id())
	tp.txlist[*c.tx.Id()] = *c

}

//convert from wire format to TxMsg
func TxMsgFromWire(tx *factomwire.MsgTx) (txm *TxMsg) {
	txm.UnmarshalBinary(tx.Data)
	return
}

//*** TxProcessor implimentaion ***//
//set context to tx, this context will be used by default when
// by Verify and AddToMemPool
// see factomd.TxMempool
func (fp *factoidPool) SetContext(tx *factomwire.MsgTx) {
	fp.context = context{
		wire: tx,
		tx:   NewTx(TxMsgFromWire(tx)),
	}
}

//Verify is designed to be called by external packages without
// them needing to know the specific Tx foramt
func (fp *factoidPool) Verify() (ret bool) {
	if !fp.utxo.IsValid(fp.context.tx.Txm.TxData.Inputs) {
		return false
	}

	if _, ok := fp.txpool.txlist[*fp.context.tx.Id()]; ok {
		return false
	}

	//ToDo: verify signatures
	if !Verify(fp.context.tx) {
		return false
	}
	return true
}

//func (*factoidPool) Broadcast()  {return}

//add transaction to memorypool after passing verification of signature and utxo
//assume the transaction is already set to factoidpool.context via call to
// SetContext
func (fp *factoidPool) AddToMemPool() {
	fp.utxo.AddTx(fp.context.tx)
	fp.txpool.AddContext(&fp.context)
	return
}

//******//
