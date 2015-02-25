// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (

	//"math"
	"bytes"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"time"
)

const (
	FaucetTxidStr = "FaucetTxidForSymulationTesting01"
)

var (
	FaucetTxid *Txid
)

func init() {
	FaucetTxid = new(Txid)
	copy(FaucetTxid[:], []byte(FaucetTxidStr))
}

func VerifyTx(tx *Tx) bool {
	//fmt.Println("Verifyxxxxx ")

	sig := tx.Txm.Sigs
	in := tx.Txm.TxData.Inputs
	sigcount := len(sig) - 1
	incount := len(in) - 1

	//fmt.Println("Verify ",sigcount,incount)
	if sigcount < 0 || incount < 0 {
		return false
	}

	si, ii := 0, 0
	for {

		if VerifyInputSig(&in[ii], &sig[si], tx) {
			ii++
			if ii > incount {
				return true
			}
		} else if si == sigcount {
			return false
		} //no more sigs - input not verified

		if si < sigcount {
			si++
		}

		//Inputs[ii] is now verified
	}

	return false
}

func VerifyInputSig(in *Input, sig *InputSig, tx *Tx) bool {
	//ToDo: remove for production
	if IsFaucet(in) {
		return true
	}

	///ToDo: MultiSig
	for i := 0; i < len(sig.Sigs); i++ {
		if notaryapi.VerifySlice(in.RevealAddr[:], tx.Digest(), sig.Sigs[i].Sig[:]) {
			return true
		}

		if i >= 2 {
			return false
		} //only allow 3 bad sigs
	}

	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func NewTxFromOutputToAddr(txid *Txid, outs []Output, outnum uint32, from AddressReveal, to Address) (txm *TxMsg) {
	//fmt.Println("NewTxFromOutputToAddr %#v", from)

	//txd := NewTxData()
	if uint32(len(outs)) < outnum || outnum < 1 {
		fmt.Println("NewTxFromOutputToAddr err1 %#v", outs)
		return nil
	}
	if ok := VerifyAddressReveal(outs[outnum-1].ToAddr, from); !ok {
		fmt.Println("NewTxFromOutputToAddr !VerifyAddressReveal %#v %#v", from, outs)
		return nil
	}

	in := NewInput(txid, outnum, from)
	return NewTxFromInputToAddr(in, outs[outnum-1].Amount, to)
}

/*
func NewTxFromOutAmountToAddr(txid *Txid, outs []Output, outnum uint32, amount int64, from AddressReveal, to Address) (txm *TxMsg) {
	fmt.Println("NewTxFromOutAmountToAddr %#v", outs)

	//txd := NewTxData()
	if uint32(len(outs)) < outnum || outnum < 1 {
		fmt.Println("NewTxFromOutputToAddr err1 %#v", outs)
		return nil
	}
	if !VerifyAddressReveal(outs[outnum-1].ToAddr, from) {
		fmt.Println("NewTxFromOutputToAddr !VerifyAddressReveal %#v %#v", from, outs)
		return nil
	}

	if amount > outs[outnum-1].Amount {
		fmt.Println("NewTxFromOutputToAddr Invalid amount %#v %#v", amount, outs)
		return nil
	}

	in := NewInput(txid, outnum, from[:])
	return NewTxFromInputToAddr(in,outs[outnum-1].Amount,to)
}
*/

func NewTxFromInputToAddr(in *Input, snowAmount int64, to Address) (txm *TxMsg) {
	//fmt.Println("NewTxFromOutputToAddr %#v", from)

	txd := NewTxData()
	txd.AddInput(*in)

	out := NewOutput(FACTOID_ADDR, snowAmount, to)
	txd.AddOutput(*out)
	return NewTxMsg(txd)
}

func VerifyAddressReveal(address Address, reveal AddressReveal) bool {
	hash := notaryapi.Sha(reveal[:])

	if bytes.Compare(address, hash.Bytes) == 0 {
		return true
	}

	return false
}

func OutputsTx(tx *Tx) []Output {
	return tx.Txm.TxData.Outputs
}

func AddSingleSigToTxMsg(txm *TxMsg, ss *SingleSignature) {
	is := InputSig{}
	is.AddSig(*ss)

	txm.AddInputSig(is)
}

//Create Input with faucet TXID. This input will not be checked against Utxo
// and will always be valid. Nonce is used to make resulting Transactions unique
// only need to update nonce when rest to Tx is same a previous one.
///ToDo: remove before production
func NewFaucetInput(nonce uint32) *Input {
	return NewInput(FaucetTxid, nonce, AddressReveal{})
}

//use time as nonce - shouls always work
func NewFaucetIn() *Input {
	return NewInput(FaucetTxid, uint32(time.Now().Unix()), AddressReveal{})
}

func IsFaucet(in *Input) bool {
	return in.Txid == *FaucetTxid
}

//------------------------------------------------
// FBlock array sorting implementation - accending
type ByFBlockIDAccending []FBlock

func (f ByFBlockIDAccending) Len() int {
	return len(f)
}
func (f ByFBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.Height < f[j].Header.Height
}
func (f ByFBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
