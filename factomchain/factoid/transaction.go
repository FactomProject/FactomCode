// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	//"bytes"
	//"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"	
)

type Txid notaryapi.HashF

//Input is the UTXO being spent
//	Txid defines the transaction that contains Output being spent
//	Index is Output number 
//	RevealAddr is the "reveal" for the Output.ToAddr "commit"
//			it contains the public-key(s) corresponding to Signatures 
type Input struct {
	Txid			Txid 
	Index			uint32
	RevealAddr		[]byte
}

//Output defines a receiver of the Input
//	Output.Type 
//		FACTOID_ADDR
//		ENTRYCREDIT_PKEY
//
//	Amount is amount of transfer in "Snow"
//	ToAddr is a hash of struct, with public-keys(s)
//		for Type = ENTRYCREDIT_PKEY , ToAddr is public-key 
type Output struct {
	Type 			byte
	Amount			int64
	ToAddr		 	Address	
}

//TxData is the core of the transaction, it generates the TXID
//TxData is signed by each input 
//	LockTime is intened as used in bitcoin
type TxData struct {
	Inputs     	[]Input
	Outputs    	[]Output
	LockTime 	uint32
}

//TxMsg is the signed and versioned Factoid transaction message
//	Sigs is at least 1 signature per Input
type TxMsg struct {
	Version  	int32
	TxData		*TxData
	Sigs		[]InputSig
}

//Tx is the TxMsg and a chache of its Txid 
type Tx struct {
	Raw  		*TxMsg
	id			*Txid
}

//return transaction id of transacion
func (td *TxData) Txid() Txid {
	var txid Txid
	return txid
}

//return transaction id of transacion
func (txm *TxMsg) Txid() Txid {
	return txm.TxData.Txid()
}

// NewTx returns a new instance of a bitcoin transaction given an underlying
// btcwire.MsgTx.  See Tx.
func NewTx(wire *TxMsg) *Tx {
	return &Tx{
		Raw:	wire,
		id: 	nil,
	}
}

func (tx *Tx) Id() Txid {
	if tx.id == nil { 
		txid := tx.Raw.Txid()
		tx.id = &txid
	}

	return *tx.id
}

type TxSpentList []bool
var Utxo map[Txid]TxSpentList  

