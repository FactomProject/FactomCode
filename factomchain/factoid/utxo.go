// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
//"bytes"
//"fmt"
//"github.com/FactomProject/FactomCode/notaryapi"
)

//ToDo: use bits if faster , or if keeping full utxo in memory. should really do memory mapped files//
//
//TxSpentList is a vector of bool indicators of:
//	if $Txids[tx.Input[i].Txid].Output[tx.Input[i].Index] have been spent
//
//Output type ENTRYCREDIT_PKEY outputs are always flagged as spent
//
type TxSpentList []bool

//Unspent TransaXtion Outputs is implimented as a hashtable from Txid to bool array  TxSpentList
type Utxo struct {
	Txspent map[Txid]TxSpentList
}

//Utxo constructor
func NewUtxo() Utxo {
	return Utxo{
		Txspent: make(map[Txid]TxSpentList),
	}
}

//add transaction to Utxo
//used when tx passed all verifications
//ToDo: maybe support Udue (command pattern?)
func (u *Utxo) AddTx(t *Tx) {
	_, ok := u.Txspent[t.Id()]
	if ok {
		return
	}

	if !u.IsValid(t.Raw.TxData.Inputs) {
		return
	}
	txs := make(TxSpentList, len(t.Raw.TxData.Outputs)+1)

	u.Txspent[t.Id()] = txs
	u.Spend(t.Raw.TxData.Inputs)
}

//IsValid checks Inputs and returns true if all inputs are in current Utxo
func (u *Utxo) IsValid(in []Input) bool {
	for _, i := range in {
		spends, ok := u.Txspent[i.Txid]
		if !ok {
			return false
		}

		if i.Index < 1 || i.Index >= uint32(len(spends)) {
			return false
		}
		if spends[i.Index-1] {
			return false
		}
	}

	return true
}

//Spend will mark Input as Spent in Utxo
func (u *Utxo) Spend(in []Input) {
	for _, i := range in {
		spends, ok := u.Txspent[i.Txid]
		if !ok {
			continue
		}

		if i.Index < 1 || i.Index >= uint32(len(spends)) {
			continue
		}
		if !spends[i.Index-1] {
			spends[i.Index-1] = true
		}
	}
}
