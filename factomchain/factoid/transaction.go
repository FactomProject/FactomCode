// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	//"bytes"
	//"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"	
)

//Input is the UTXO being spent
//	Txid defines the transaction that contains Output being spent
//	Index is Output number 
//	SigData is the "reveal" for the Output.Address "commit"
//			it contains the public-key(s) corresponsong to Signatures 
type Input struct {
	Txid		notaryapi.Hash 
	Index		uint32
	SigData		[]byte
}

//Output defines a receiver of the Input
//	Output.Type 
//		FACTOID_ADDR
//		ENTRYCREDIT_PKEY
//
//	Amount is amount of transfer in "Snow"
//	Address is a hash of struct, with public-keys(s)
//		for Type = ENTRYCREDIT_PKEY , address is public-key 
type Output struct {
	Type 		byte
	Amount		int64
	Address 	notaryapi.Hash		
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
//	Sig is at least 1 signature per Input
type TxMsg struct {
	Version  	int32
	TxData		*TxData
	Sig			[][]byte
}