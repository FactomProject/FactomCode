// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"sync"
		"github.com/FactomProject/FactomCode/factomwire"

)


// txMemPool is used as a source of transactions that need to be mined into
// blocks and relayed to other peers.  It is safe for concurrent access from
// multiple peers.
type factoidPool struct {
	sync.RWMutex
	//server        *server
	utxo 		Utxo
	context 	*factomwire.MsgTx
	tx 			*Tx
}


// newTxMemPool returns a new memory pool for validating and storing standalone
// transactions until they are mined into a block.
func NewFactoidPool() *factoidPool {
	return &factoidPool{
		//server:        server,
		utxo:          NewUtxo(),
	}
}

func (fp *factoidPool) SetContext(tx *factomwire.MsgTx) {
	fp.context = tx

	txm := new(TxMsg)
	txm.UnmarshalBinary(tx.Data)
	fp.tx = NewTx( txm )
}
func (fp *factoidPool) Verify() (ret bool) {
	ret = fp.utxo.IsValid(fp.tx.Raw.TxData.Inputs)
	return true
}

//func (*factoidPool) Broadcast()  {return}
func (fp *factoidPool) AddToMemPool()  {
	fp.utxo.AddTx(fp.tx)
	return
}

