package factoid_test

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"	
	"testing"
	"github.com/FactomProject/FactomCode/wallet"	
	"github.com/FactomProject/FactomCode/notaryapi"	

)	


func TestECAddress(t *testing.T) {
	var mypub = wallet.ClientPublicKey()

	addr := factoid.EncodeAddress((*mypub.Key)[:],'\x09')
	t.Logf("addr: %v", addr)
}

func TestFactoidAddress(t *testing.T) {
	var mypub = wallet.ClientPublicKey()

	hash := notaryapi.Sha((*mypub.Key)[:])

	addr := factoid.EncodeAddress(hash.Bytes,'\x09')
	t.Logf("addr: %v len %v", addr, len(addr))

}
