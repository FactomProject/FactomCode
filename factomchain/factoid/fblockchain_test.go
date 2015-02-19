package factoid_test

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
	"testing"
	"github.com/FactomProject/FactomCode/wallet"
	//"github.com/FactomProject/FactomCode/notaryapi"

)

func TestFactoidGenesis(t *testing.T) {
	genb := factoid.FactoidGenesis(factomwire.TestNet)

	t.Logf("genb: %v ",genb)

	t.Logf("txid: %v ",genb.Transactions[0].Id())


	t.Logf("len: %v ",len(genb.Transactions[0].Raw.Sigs))

	ok := wallet.ClientPublicKey().Verify(genb.Transactions[0].Digest(),(*[64]byte)(&genb.Transactions[0].Raw.Sigs[0].Sigs[0].Sig))
	//ok := wallet.ClientPublicKey().Verify(genb.Transactions[0].Digest(),(&genb.Transactions[0].Raw.Sigs[0].Sigs[0].Sig))

	if !ok { t.Fatalf("!Verify")}
}
	
