// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	//"bytes"
	"fmt"
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

type TxOutSpent struct {
	Outs  []Output
	Spent TxSpentList
}

//Unspent TransaXtion Outputs is implimented as a hashtable from Txid to bool array  TxSpentList
type Utxo struct {
	Txspent map[Txid]TxOutSpent
}

//Utxo constructor
func NewUtxo() Utxo {
	return Utxo{
		Txspent: make(map[Txid]TxOutSpent),
	}
}

//add transaction to Utxo
//used when tx passed all verifications
//ToDo: maybe support Udue (command pattern?)
func (u *Utxo) AddTx(t *Tx) {
	fmt.Println("AddTx", t.Id().String())

	_, ok := u.Txspent[*t.Id()]
	if ok {
		return
	}

	if !u.IsValid(t.Txm.TxData.Inputs) {
		return
	}

	u.AddUtxo(t.Id(), t.Txm.TxData.Outputs)
	u.Spend(t.Txm.TxData.Inputs)
}

//add all outputs for txid to UTXO
func (u *Utxo) AddUtxo(id *Txid, outs []Output) {
	txs := make(TxSpentList, len(outs))
	u.Txspent[*id] = TxOutSpent{outs, txs}
}

//IsValid checks Inputs and returns true if all inputs are in current Utxo
func (u *Utxo) IsValid(in []Input) bool {
	for _, i := range in {

		//ToDo: remove for production
		if IsFaucet(&i) {
			continue
		}

		spends, ok := u.Txspent[i.Txid]
		if !ok {
			fmt.Println("Txid not known", i.Txid.String())
			return false
		}

		if i.Index < 1 || i.Index > uint32(len(spends.Spent)) {
			fmt.Println("utxo bad index", i.Txid.String(), " Index ", i.Index)

			return false
		}
		if spends.Spent[i.Index-1] {
			fmt.Println("Output already spent ", i.Txid.String(), " Index ", i.Index)
			return false
		}
		if ok := VerifyAddressReveal(spends.Outs[i.Index-1].ToAddr, i.RevealAddr); !ok {
			fmt.Println("!VerifyAddressReveal ", spends.Outs[i.Index-1].ToAddr, " i.RevealAddr ", i.RevealAddr)
			return false
		}

	}

	return true
}

//true if input parents are known
func (u *Utxo) InputsKnown(in []Input) (bool, []*Txid) {
	var missing []*Txid
	for _, i := range in {
		//ToDo: remove for production
		if IsFaucet(&i) {
			continue
		}

		_, ok := u.Txspent[i.Txid]
		if !ok {
			missing = append(missing, &i.Txid)
			//fmt.Println("Txid not known", i.Txid.String())
		}
	}

	return len(missing) == 0, missing
}

//Spend will mark Input as Spent in Utxo
func (u *Utxo) Spend(in []Input) {
	for _, i := range in {
		fmt.Println("Utxo Spent", i.Txid.String(), " Index ", i.Index)
		//ToDo: remove for production
		if IsFaucet(&i) {
			continue
		}

		spends, ok := u.Txspent[i.Txid]
		if !ok {
			continue
		}

		if i.Index < 1 || i.Index > uint32(len(spends.Spent)) {
			continue
		}

		if !spends.Spent[i.Index-1] {
			spends.Spent[i.Index-1] = true
		}
	}
}

//Meant for adding verified blocks to Utxo
func (u *Utxo) AddVerifiedTxList(tx []Tx) {
	for _, t := range tx {
		u.AddUtxo(t.Id(), OutputsTx(&t))
	}
}
