// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"fmt"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"

	"sync"
)

const (
	//initail array allocation
	//each call to append will double capacity when len() = cap()
	TxPoolAllocSize = 1000
)

//GlobalUtxo will store utxo for all blocks
// temp use global var, to be called from "restapi"
// ToDo: redesign
var GlobalUtxo Utxo

func init() {
	GlobalUtxo = NewUtxo()
}

// ToDo: TxProcessor was copied from factomd.txmempool due to import cycle issues. fix this
//
// TxProcessor is an interface that abstracts methods needed to process a TxMessage
// SetContext(*TxMessage) will store to concrete TxMessage in the concrete TxProcessor object
type TxProcessor interface {
	SetContext(*factomwire.MsgTx)
	Verify() bool
	//Broadcast()
	AddToMemPool() []*notaryapi.HashF
	Confirm(*factomwire.MsgConfirmation) (bool, []*notaryapi.HashF)
}

// txMemPool is used as a source of transactions that need to be mined into
// blocks and relayed to other peers.  It is safe for concurrent access from
// multiple peers.
//   factoidpool impliments TxProcessor for generic mempool processing
type FactoidPool struct {
	TxProcessor
	sync.RWMutex
	//server        *server
	utxo       *Utxo
	context    context
	txpool     *txpool
	orphanpool *orphanpool
}

//the context is the tx used by default when no tx is referenced
// this is needed in order for mempool methods to be called from abstract interface
// when different mempools have different tx structs  (no plyorphism in go)
type context struct {
	wire *factomwire.MsgTx
	tx   *Tx
	//index   int //index into txpool txindex array of *Txid
	missing []*Txid
}

//orphanpool stores an map of orphan Txids to context
// and a map of missing parent txids to list of orphan children
type orphanpool struct {
	orphans map[Txid]context
	parents map[Txid][]*Txid
}

//create new orphanpool
func NewOrphanPool() *orphanpool {
	return &orphanpool{
		orphans: make(map[Txid]context),
		parents: make(map[Txid][]*Txid),
		//nexti: 		0
	}
}

//txpool stores an array of Txids in order of insertion and
//a map from Txid to the context tx
type txpool struct {
	//txindex []*Txid
	txlist map[Txid]context
	//nexti		int
}

//create new txpool, allocate TxPoolAllocSize
func NewTxPool() *txpool {
	return &txpool{
		//txindex: make([]*Txid, 0, TxPoolAllocSize),
		txlist: make(map[Txid]context),
		//nexti: 		0
	}
}

// newTxMemPool returns a new memory pool for validating and storing standalone
// transactions until they are mined into a block.
func NewFactoidPool() *FactoidPool {
	return &FactoidPool{
		//server:        server,
		utxo:       &GlobalUtxo,
		txpool:     NewTxPool(),
		orphanpool: NewOrphanPool(),
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
	return fp.utxo
}

//add context tx to txpool
func (tp *txpool) AddContext(c *context) {
	//c.index = len(tp.txindex)
	//tp.txindex = append(tp.txindex, c.tx.Id())
	tp.txlist[*c.tx.Id()] = *c

}

func (fp *FactoidPool) GetTxContext() *Tx {
	return fp.context.tx
}

//add context tx to orphanpool
func (op *orphanpool) AddContext(c *context) {
	op.orphans[*c.tx.Id()] = *c
	for _, id := range c.missing {
		op.parents[*id] = append(op.parents[*id], c.tx.Id())
	}
}

//add context tx to orphanpool
func (op *orphanpool) FoundMissing(txid *Txid) (children []*Txid, ok bool) {
	children, ok = op.parents[*txid]
	return
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
	rtx := NewTx(TxMsgFromWire(tx))
	fp.context = context{
		wire: tx,
		tx:   rtx,
	}
}

//Verify is designed to be called by external packages without
// them needing to know the specific Tx foramt
func (fp *FactoidPool) Verify() (ret bool) {
	ok, parents := fp.utxo.InputsKnown(fp.context.tx.Txm.TxData.Inputs)

	if ok { // parents are known
		if !fp.utxo.IsValid(fp.context.tx.Txm.TxData.Inputs) {
			fmt.Println("!fp.utxo.IsValid")
			return false
		}

		if _, ok := fp.txpool.txlist[*fp.context.tx.Id()]; ok {
			fmt.Println("!Verify: tx already exists in fp.txpool.txlist")
			return false
		}
	} else { // is orphan

		if _, ok := fp.orphanpool.orphans[*fp.context.tx.Id()]; ok {
			fmt.Println("!Verify: orphan tx already exists in fp.orphanpool.orphans")
			return false
		}

		if len(parents) == 0 {
			fmt.Println("!Verify: is orphan but no missing parents - should not happen")
			return false
		}
		//ToDo: max orphan size
		//fp.context.isorphan = true;
		fp.context.missing = parents
	}

	ok = VerifyTx(fp.context.tx)
	//verify signatures
	if !ok {
		fmt.Println("!Verify sigs", fp.context.tx, VerifyTx)

		return false
	}
	return true
}

//func (*FactoidPool) Broadcast()  {return}

//add transaction to memorypool after passing verification of signature and utxo
//assume the transaction is already set to factoidpool.context via call to
// SetContext
func (fp *FactoidPool) AddToMemPool() (verified []*notaryapi.HashF) {
	fmt.Println("AddToMemPool", fp.context.tx.Id().String())

	if len(fp.context.missing) > 0 { // is orphan
		fp.orphanpool.AddContext(&fp.context)
	} else {
		verified = append(verified, fp.doAdd(&fp.context))
		//see if this tx is a missing parent of orphan[s]
		if kids, ok := fp.orphanpool.FoundMissing(fp.context.tx.Id()); ok {
			//for each orphan child of parent
			for _, k := range kids {
				ocontext := fp.orphanpool.orphans[*k]
				//see if orphan is still missing more parents
				ok2, missing := fp.utxo.InputsKnown(ocontext.tx.Txm.TxData.Inputs)
				if ok2 { //all parents found
					if fp.utxo.IsValid(ocontext.tx.Txm.TxData.Inputs) {
						ocontext.missing = missing
						verified = append(verified, fp.doAdd(&ocontext))
					}
					delete(fp.orphanpool.orphans, *k)
				} else { // still orphan
					copy(fp.orphanpool.orphans[*k].missing[:], missing)
				}
			}

			//remove parent from missing list
			delete(fp.orphanpool.parents, *fp.context.tx.Id())
		}
	}

	return
}

func (fp *FactoidPool) doAdd(c *context) *notaryapi.HashF {
	fp.utxo.AddTx(c.tx)
	fp.txpool.AddContext(c)

	return (*notaryapi.HashF)(c.tx.Id())
}

//Confirm is called when receivedd a confirmation msg from federatd server
// should only be called after already received the Tx
// ToDo: deal with confirms that come in diff order than in pool,
//		may need to store doubel spends, they may become valid
func (fp *FactoidPool) Confirm(fm *factomwire.MsgConfirmation) (ok bool, confirmed []*notaryapi.HashF) {
	if cx, ok := fp.txpool.txlist[fm.Affirmation]; ok {
		confirmed = append(confirmed, (*notaryapi.HashF)(cx.tx.Id()))
		delete(fp.txpool.txlist, fm.Affirmation)
	}
	return
}

//******//
