package factoid_test

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/wallet"
	"testing"
	//"fmt"
)

func TestMemPoolGenesis(t *testing.T) {
	fp := factoid.NewFactoidPool()
	fp.AddGenesisBlock()

	t.Logf("%#v ", *fp)

	t.Logf("%#v ", *fp.Utxo())

}

func TestMemPoolGenesisSpend(t *testing.T) {
	t.Logf("herer ")

	genb := factoid.FactoidGenesis(factomwire.TestNet)

	addr, _, _ := factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSs")
	outs := factoid.OutputsTx(&genb.Transactions[0])
	txm := factoid.NewTxFromOutputToAddr(genb.Transactions[0].Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)

	ds := wallet.DetachMarshalSign(txm.TxData)
	ss := factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)
	/*
		t.Logf("%#v",txm.TxData.Outputs)

		t.Logf("%#v",txm.TxData.Inputs)

		tx := factoid.NewTx(txm)

		ok := factoid.VerifyTx(tx)
		if !ok { t.Fatalf("!Verify")}
	*/
	fp := factoid.NewFactoidPool()
	fp.AddGenesisBlock()

	t.Logf("%#v", fp)

	wire := factoid.TxMsgToWire(txm)
	t.Logf("%#v", wire)
	/*
		txm2 := factoid.TxMsg{}
		txm2.UnmarshalBinary(wire.Data)
	*/
	t.Logf("%#v ", *fp.Utxo())

	fp.SetContext(wire)
	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	fp.SetContext(wire)
	if ok := fp.Verify(); ok {
		t.Fatalf("VerifyOk double spend")
	}

	/*
		fmt.Println("tmx1 %v",txm.TxData.Inputs[0].RevealAddr)

		inp := txm.TxData.Inputs[0]
		fmt.Println("inp ",inp.String())

		minp , err := inp.MarshalBinary()
		fmt.Println("minp",err, minp, len(minp))

		inp.UnmarshalBinary(minp)
		fmt.Println("inp ",inp.String())

		outp := txm.TxData.Outputs[0]
		fmt.Println("outp ",outp.String())

		mbo , err := outp.MarshalBinary()
		fmt.Println("mbo",err, mbo, len(mbo))

		outp.UnmarshalBinary(mbo)
		fmt.Println("outp2 ",outp.String())


		txd := txm.TxData
		fmt.Println("txd %#v",txd)

		tmbo , err := txd.MarshalBinary()
		fmt.Println("tmbo",err, tmbo, len(tmbo))

		txd.UnmarshalBinary(tmbo)
		fmt.Println("txd2 %#v",txd)

		outp = txm.TxData.Outputs[0]
		t.Logf("outp %#v",outp.String())

		wire := factoid.TxMsgToWire(txm)
		t.Logf("wire %#v ", *wire)

		txm = factoid.TxMsgFromWire(wire)
		t.Logf("txm %#v ", *txm)

		fmt.Println("here1")

		fp := factoid.NewFactoidPool()
		fp.AddGenesisBlock()
		t.Logf("%#v ", *fp.Utxo())

		fp.SetContext(wire)
		if ok := fp.Verify(); !ok {
			t.Fatalf("!Verify")
		}

		fp.AddToMemPool()
	*/
}
