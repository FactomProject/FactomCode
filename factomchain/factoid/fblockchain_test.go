package factoid_test

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/wallet"
	"testing"
	//"github.com/FactomProject/FactomCode/notaryapi"
)

func TestFactoidGenesis(t *testing.T) {
	if wallet.FactoidAddress() != factoid.GenesisAddress {
		t.SkipNow()
	}

	genb := factoid.FactoidGenesis(factomwire.TestNet)

	t.Logf("genb: %v ", genb)

	t.Logf("txid: %v ", genb.Transactions[0].Id())

	t.Logf("len: %v ", len(genb.Transactions[0].Txm.Sigs))

	ok := wallet.ClientPublicKey().Verify(genb.Transactions[0].Digest(), (*[64]byte)(&genb.Transactions[0].Txm.Sigs[0].Sigs[0].Sig))
	//ok := wallet.ClientPublicKey().Verify(genb.Transactions[0].Digest(),(&genb.Transactions[0].Raw.Sigs[0].Sigs[0].Sig))

	if !ok {
		t.Fatalf("!Verify")
	}
}

func TestGenesisInputTransaction(t *testing.T) {
	if wallet.FactoidAddress() != factoid.GenesisAddress {
		t.SkipNow()
	}

	genb := factoid.FactoidGenesis(factomwire.TestNet)

	addr, _, _ := factoid.DecodeAddress("ExZ7hUZ7B4T3doVC6iLBPh9JP33huwELmLg6pM2LDNSiqk9mSs")
	outs := factoid.OutputsTx(&genb.Transactions[0])
	txm := factoid.NewTxFromOutputToAddr(genb.Transactions[0].Id(), outs, uint32(1), factoid.AddressReveal(*wallet.ClientPublicKey().Key), addr)

	t.Logf("%#v ", txm)
	t.Logf("%#v ", txm.TxData)

	ds := wallet.DetachMarshalSign(txm.TxData)
	ss := factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	if !factoid.VerifyTx(factoid.NewTx(txm)) {
		t.Fatalf("!Verify")
	}

}
