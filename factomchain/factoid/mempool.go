// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"fmt"
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
type FactoidPool struct {
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
func NewFactoidPool() *FactoidPool {
	return &FactoidPool{
		//server:        server,
		utxo:   NewUtxo(),
		txpool: NewTxPool(),
	}
}

func (fp *FactoidPool) AddGenesisBlock() {
	genb := FactoidGenesis(factomwire.TestNet)

	c := context{
		wire: nil,
		tx:   &genb.Transactions[0],
	}
	fp.txpool.AddContext(&c)

	fp.utxo.AddUtxo(c.tx.Id(), c.tx.Txm.TxData.Outputs)

}

func (fp *FactoidPool) Utxo() *Utxo {
	return &fp.utxo
}

//add context tx to txpool
func (tp *txpool) AddContext(c *context) {
	c.index = len(tp.txindex)
	tp.txindex = append(tp.txindex, c.tx.Id())
	tp.txlist[*c.tx.Id()] = *c

}

//convert from wire format to TxMsg
func TxMsgFromWire(tx *factomwire.MsgTx) (txm *TxMsg) {
	txm = new(TxMsg)
	txm.UnmarshalBinary(tx.Data)
	//fmt.Println("---%#v",txm)
	return
}

//convert from TxMsg to wire format
func TxMsgToWire(txm *TxMsg) (tx *factomwire.MsgTx) {
	tx = new(factomwire.MsgTx)
	tx.Data, _ = txm.MarshalBinary()
	return
}

//*** TxProcessor implimentaion ***//
//set context to tx, this context will be used by default when
// by Verify and AddToMemPool
// see factomd.TxMempool
func (fp *FactoidPool) SetContext(tx *factomwire.MsgTx) {
	fp.context = context{
		wire: tx,
		tx:   NewTx(TxMsgFromWire(tx)),
	}
}

//Verify is designed to be called by external packages without
// them needing to know the specific Tx foramt
func (fp *FactoidPool) Verify() (ret bool) {

	if !fp.utxo.IsValid(fp.context.tx.Txm.TxData.Inputs) {
		fmt.Println("!fp.utxo.IsValid")
		return false
	}

	if _, ok := fp.txpool.txlist[*fp.context.tx.Id()]; ok {
		fmt.Println("fp.txpool.txlist")
		return false
	}

	ok := VerifyTx(fp.context.tx)
	//verify signatures
	if !ok {
		fmt.Println("sigs", fp.context.tx, VerifyTx)

		return false
	}
	return true
}

//func (*FactoidPool) Broadcast()  {return}

//add transaction to memorypool after passing verification of signature and utxo
//assume the transaction is already set to factoidpool.context via call to
// SetContext
func (fp *FactoidPool) AddToMemPool() {
	fmt.Println("AddToMemPool", fp.context.tx.Id().String())

	fp.utxo.AddTx(fp.context.tx)
	fp.txpool.AddContext(&fp.context)
	return
}

//******//
