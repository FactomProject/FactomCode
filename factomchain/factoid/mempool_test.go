package factoid_test

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	//"github.com/FactomProject/FactomCode/factomwire"
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

func TestFaucet(t *testing.T) {
	//t.Logf("%#v ", factoid.FaucetTxidStr)

	//b := []byte(factoid.FaucetTxidStr)
	//t.Logf("%#v ", b)

	//t.Logf("txidstr %#v ", factoid.FaucetTxid.String())

	//in := factoid.NewFaucetInput(0)
	//t.Logf("%#v ", in)

	//if !factoid.IsFaucet(in) {
	//	t.Fatalf("IsFaucet")
	//}

	//add 1000 factoids to my address using faucet
	fp := factoid.NewFactoidPool()
	addr, _, _ := factoid.DecodeAddress(wallet.FactoidAddress())
	txm := factoid.NewTxFromInputToAddr(factoid.NewFaucetInput(0), 1000, addr)
	ds := wallet.DetachMarshalSign(txm.TxData)
	ss := factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	if !factoid.IsFaucet(&txm.TxData.Inputs[0]) {
		t.Fatalf("!IsFaucet")
	}

	utxo := factoid.NewUtxo()
	if !utxo.IsValid(txm.TxData.Inputs) {
		t.Fatalf("!IsValidxx")
	}

	wire := factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	tx := fp.GetTxContext()

	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify1")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	//spend to external address
	prevtx := tx
	addr, _, _ = factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSx")
	outs := factoid.OutputsTx(prevtx)
	txm = factoid.NewTxFromOutputToAddr(prevtx.Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify2")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	//try again - should fail
	//prevtx = factoid.NewTx(txm)
	addr, _, _ = factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSx")
	outs = factoid.OutputsTx(prevtx)
	txm = factoid.NewTxFromOutputToAddr(prevtx.Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	if ok := fp.Verify(); ok {
		t.Fatalf("should fail Verify3")
	}

	//try new faucet - should fail with nonce0
	//add 1000 factoids to my address using faucet
	addr, _, _ = factoid.DecodeAddress(wallet.FactoidAddress())
	txm = factoid.NewTxFromInputToAddr(factoid.NewFaucetInput(0), 1000, addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	tx = fp.GetTxContext()
	if ok := fp.Verify(); ok {
		t.Fatalf("Verify should fail")
	}

	//fp.AddToMemPool()
	//t.Logf("%#v ", *fp.Utxo())

	//try new faucet - should pass with nonce1
	//add 1000 factoids to my address using faucet
	addr, _, _ = factoid.DecodeAddress(wallet.FactoidAddress())
	txm = factoid.NewTxFromInputToAddr(factoid.NewFaucetInput(1), 1000, addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	//t.Logf("1 before fail %#v ")

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	tx = fp.GetTxContext()
	t.Logf("1 before fail %#v ", tx.Id().String())

	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify nonce1")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	//inn := factoid.NewFaucetIn()
	//t.Logf("infa %d ", inn.Index)

	//try new faucet - should pass with (using time for nonce)
	//add 1000 factoids to my address using faucet
	addr, _, _ = factoid.DecodeAddress(wallet.FactoidAddress())
	txm = factoid.NewTxFromInputToAddr(factoid.NewFaucetIn(), 1000, addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	tx = fp.GetTxContext()
	t.Logf("before fail %#v ", txm.TxData.Inputs[0])
	t.Logf("1 before fail %#v ", tx.Id().String())

	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	//try sending from faucet tx input
	//spend to my wallet address
	prevtx = tx
	addr, _, _ = factoid.DecodeAddress(wallet.FactoidAddress())
	outs = factoid.OutputsTx(prevtx)
	txm = factoid.NewTxFromOutputToAddr(prevtx.Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	tx = fp.GetTxContext()
	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify 4")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	//try sending from prev tx input
	//spend to an external address
	prevtx = tx
	addr, _, _ = factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSx")
	outs = factoid.OutputsTx(prevtx)
	txm = factoid.NewTxFromOutputToAddr(prevtx.Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	if ok := fp.Verify(); !ok {
		t.Fatalf("!Verify 5")
	}

	fp.AddToMemPool()
	t.Logf("%#v ", *fp.Utxo())

	//try sending from prev tx input
	//should fail - bad sig
	prevtx = tx
	addr, _, _ = factoid.DecodeAddress(wallet.FactoidAddress())
	outs = factoid.OutputsTx(prevtx)
	txm = factoid.NewTxFromOutputToAddr(prevtx.Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)
	ds = wallet.DetachMarshalSign(txm.TxData)
	ss = factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire = factoid.TxMsgToWire(txm)
	fp.SetContext(wire)
	if ok := fp.Verify(); ok {
		t.Fatalf("!Should fail 5")
	}

	//fp.AddToMemPool()
	//t.Logf("%#v ", *fp.Utxo())

}

/*
func TestMemPoolGenesisSpend(t *testing.T) {
	t.Logf("herer ")

	genb := factoid.FactoidGenesis(factomwire.TestNet)

	addr, _, _ := factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSs")
	outs := factoid.OutputsTx(&genb.Transactions[0])
	txm := factoid.NewTxFromOutputToAddr(genb.Transactions[0].Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)

	ds := wallet.DetachMarshalSign(txm.TxData)
	ss := factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	fp := factoid.NewFactoidPool()
	fp.AddGenesisBlock()

	t.Logf("%#v", fp)

	wire := factoid.TxMsgToWire(txm)
	t.Logf("%#v", wire)

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

}

*/
