// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid
import (
	//"bytes"
	//"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomwire"
	"encoding/hex"

)


//Factoid Block header 
type FBlockHeader struct {
	Height			uint64
	ParentHash	 	notaryapi.HashF
	TimeStamp     	int64
	TxCount	    	uint32
}

//Factoid Block - contains list of Tx, which has raw MsgTx plus the Txid 
type FBlock struct {
	Header 			FBlockHeader
	Transactions	[]Tx
}

var genesis	*FBlock

func FactoidGenesis(net factomwire.FactomNet) (genesis *FBlock) {
	genesis = &FBlock{
		Header: FBlockHeader{
				Height:	0,
				TimeStamp: 1424305819, //Thu, 19 Feb 2015 00:30:19 GMT
				TxCount: 1,
		},
		Transactions: make([]Tx,1),
	}	

	var td TxData
	var in Input //blank input
	td.AddInput(in) 

	var out *Output
	var insig InputSig
	switch net {
		case factomwire.MainNet:

		
		default: //TestNet
			addr, _, _ := DecodeAddress("FfZgRRHxuzsWkhXcb5Tb16EYuDEkbVCPAk1svfmYxyUXGPoS2X")
			out = NewOutput(FACTOID_ADDR,1000000000,addr)
			sigbytes, _:= hex.DecodeString("4369be19d8fe9cba655cadeb4441b646b94582d0dc890bdd2060220b61bdee10b9f96cce824c201151131b99df7201f6669e9fbd9ccc74c229c57c59ed37b100")		
			insig.AddSig(SingleSigFromByte(sigbytes))

	}

	td.AddOutput(*out)
	txmsg := NewTxMsg(&td)
	txmsg.AddInputSig(insig)

	genesis.Transactions[0] = *NewTx(txmsg)
	return
}


