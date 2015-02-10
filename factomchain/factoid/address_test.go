package factoid_test

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/wallet"
	"testing"
)

func TestECAddress(t *testing.T) {
	var mypub = wallet.ClientPublicKey()

	addr := factoid.EncodeAddress((*mypub.Key)[:], '\x09')
	t.Logf("addr: %v", addr)
}

func TestFactoidAddress(t *testing.T) {
	var mypub = wallet.ClientPublicKey()

	hash := notaryapi.Sha((*mypub.Key)[:])

	addr := factoid.EncodeAddress(hash.Bytes, '\x09')
	t.Logf("addr: %v len %v", addr, len(addr))

}
