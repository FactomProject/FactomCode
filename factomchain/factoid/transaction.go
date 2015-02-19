// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	//"bytes"
	//"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"encoding/hex"
)

type Txid notaryapi.HashF

//Input is the UTXO being spent
//	Txid defines the transaction that contains Output being spent
//	Index is Output number
//	RevealAddr is the "reveal" for the Output.ToAddr "commit"
//			it contains the public-key(s) corresponding to Signatures
type Input struct {
	Txid       Txid
	Index      uint32
	RevealAddr notaryapi.ByteArray
}

// NewInput returns a new factoid transaction input with the provided
// Txid, index, and revealed address 
func NewInput(id *Txid, index uint32, reveal notaryapi.ByteArray) *Input {
	return &Input{
		Txid: *id,
		Index:  index,
		RevealAddr:	reveal,
	}
}

// Constants for Output.Type 
const (
	FACTOID_ADDR = byte(0)
	ENTRYCREDIT_PKEY = byte(1)
)

//Output defines a receiver of the Input
//	Output.Type
//		FACTOID_ADDR
//		ENTRYCREDIT_PKEY
//
//	Amount is amount of transfer in "Snow"
//	ToAddr is a hash of struct, with public-keys(s)
//		for Type = ENTRYCREDIT_PKEY , ToAddr is public-key
type Output struct {
	Type   byte
	Amount int64
	ToAddr Address
}

// NewOutput returns a new bitcoin transaction output with the provided
// transaction value and public key script.
func NewOutput(ty byte, amount int64, to Address) *Output {
	return &Output{
		Type:  	ty,
		Amount: amount,
		ToAddr: to,
	}
}

//TxData is the core of the transaction, it generates the TXID
//TxData is signed by each input
//	LockTime is intened as used in bitcoin
type TxData struct {
	Inputs   []Input
	Outputs  []Output
	LockTime uint32
}

// AddInput adds a transaction input.
func (td *TxData) AddInput(in Input) {
	td.Inputs = append(td.Inputs, in)
}

// AddOutput adds a transaction output.
func (td *TxData) AddOutput(out Output) {
	td.Outputs = append(td.Outputs, out)
}

//TxMsg is the signed and versioned Factoid transaction message
//	Sigs is at least 1 signature per Input
type TxMsg struct {
	//Version int32
	TxData  *TxData
	Sigs    []InputSig
}

//crate TxMsg from TxData
func NewTxMsg(td *TxData) *TxMsg {
	return &TxMsg{
		TxData: td,
		//Sigs: 	make([]InputSig,1),
	}
}

// AddInputSig adds a signature to transaction.
func (tm *TxMsg) AddInputSig(is InputSig) {
	tm.Sigs = append(tm.Sigs, is)
}


//Tx is the TxMsg and a chache of its Txid
type Tx struct {
	Raw *TxMsg
	id  *Txid
}

//return transaction id of transacion
func (td *TxData) Txid() (txid *Txid) {
	//var txid Txid
	txid = new(Txid)
	h, _ := notaryapi.CreateHash(td)
	(*notaryapi.HashF)(txid).From(h)
	return txid
}

//return transaction id of transacion
func (txm *TxMsg) Txid() *Txid {
	return txm.TxData.Txid()
}

// NewTx returns a new instance of a factoid transaction given an underlying
// TxMsg
func NewTx(wire *TxMsg) *Tx {
	return &Tx{
		Raw: wire,
		id:  nil,
	}
}

func (tx *Tx) Id() *Txid {
	if tx.id == nil {
		txid := tx.Raw.Txid()
		tx.id = txid
	}

	return tx.id
}

func (tx *Tx) Digest() (bin []byte) {
	bin, _ =  tx.Raw.TxData.MarshalBinary()
	return
}


func (txid *Txid) String() string {
	return hex.EncodeToString(txid[:])
}
