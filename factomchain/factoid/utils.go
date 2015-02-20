// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (

	//"math"
	"bytes"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
)

/*
func Verify(tx Tx) bool {
	sig := tx.Txm.Sigs
	in := tx.Txm.TxData.Inputs
	siglen := len(sig)
	inlen := len(in)
	if siglen == 0 || inlen == 0 { return false }

	for si, ii := 0; ii < inlen; ii++ {

		//for each Input must Verify number of RequiredSigs
		for inii := in[ii], m := 0; m < inii.RequiredSigs(); {

			moresigs := bool(si < siglen-1)
			if !VerifyInputSig(inii,sig[si]) { //signature does not match Input address
				if  !moresigs { return false } 	//no more sigs - input not verified
			}

			else { //signature verified to Input address.
				m++ //stop at m of n
			}

			if moresigs { si++ }
			//else if nomoresigs, try last sig for rest of inputs
		}

		//Inputs[ii] is now verified
	}

	return false;
}
*/

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
	///ToDo: MultiSig
	for i := 0; i < len(sig.Sigs); i++ {
		if notaryapi.VerifySlice(in.RevealAddr, tx.Digest(), sig.Sigs[i].Sig[:]) {
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

	txd := NewTxData()
	if uint32(len(outs)) < outnum || outnum < 1 {
		fmt.Println("NewTxFromOutputToAddr err1 %#v", outs)
		return nil
	}
	if !VerifyAddressReveal(outs[outnum-1].ToAddr, from) {
		fmt.Println("NewTxFromOutputToAddr !VerifyAddressReveal %#v %#v", from, outs)
		return nil
	}

	in := NewInput(txid, outnum, from[:])
	txd.AddInput(*in)

	out := NewOutput(FACTOID_ADDR, outs[outnum-1].Amount, to)
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
