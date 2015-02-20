package factoid_test

import (
	"encoding/hex"
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/wallet"
	"testing"
)

func TestECAddress(t *testing.T) {
	var mypub = wallet.ClientPublicKey()

	pk := (*mypub.Key)[:]

	netid := byte('\x14')
	addr := factoid.EncodeAddress(pk, netid)
	t.Logf("addr: %v len %v", addr, len(addr))

	decoded, ver, err := factoid.DecodeAddress(addr)

	t.Logf("pk: %v decoded %v ver %v len %v err %v", pk, decoded, ver, len(decoded), err)

	if len(pk) != len(decoded) {
		t.Fatalf("len(pk) != len(decoded)")
	}

	for i := range pk {
		if pk[i] != decoded[i] {
			t.Fatalf("pk != decoded")
		}
	}

	if ver != netid {
		t.Fatalf("ver != netid")
	}
}

func TestFactoidAddress(t *testing.T) {
	var mypub = wallet.ClientPublicKey()

	hash := notaryapi.Sha((*mypub.Key)[:])

	netid := byte('\x07')
	addr := factoid.EncodeAddress(hash.Bytes, netid)
	t.Logf("addr: %v len %v", addr, len(addr))

	decoded, ver, err := factoid.DecodeAddress(addr)

	t.Logf("hash: %v decoded %v ver %v len %v err %v", hash.Bytes, decoded, ver, len(decoded), err)

	if len(hash.Bytes) != len(decoded) {
		t.Fatalf("len(hash.Bytes) != len(decoded)")
	}

	for i := range hash.Bytes {
		if hash.Bytes[i] != decoded[i] {
			t.Fatalf("hash.Bytes != decoded")
		}
	}

	if ver != netid {
		t.Fatalf("ver != netid")
	}

}

func TestTrans(t *testing.T) {
	var td factoid.TxData
	var in factoid.Input //blank input
	td.AddInput(in)

	addr, _, _ := factoid.DecodeAddress("FfZgRRHxuzsWkhXcb5Tb16EYuDEkbVCPAk1svfmYxyUXGPoS2X")
	out := factoid.NewOutput(factoid.FACTOID_ADDR, 1000000000, addr)
	td.AddOutput(*out)

	txid := td.Txid(nil)
	//hash, err := notaryapi.CreateHash(&td)
	//(*notaryapi.HashF)(txid).From(hash)
	//copy(txid[:],hash.Bytes)
	t.Logf("txid: %v ", txid)
	//	t.Logf("txid: %v lenin %v lenout %v hash %v err %v",txid,len(td.Inputs),len(td.Outputs),hash,err)

	ds := wallet.DetachMarshalSign(&td)
	ss := factoid.NewSingleSignature(ds)
	ss.Hint = '\u554a'
	t.Logf("ss : %v ", ss)

	ssb, _ := hex.DecodeString("4369be19d8fe9cba655cadeb4441b646b94582d0dc890bdd2060220b61bdee10b9f96cce824c201151131b99df7201f6669e9fbd9ccc74c229c57c59ed37b100")
	var ss2 factoid.SingleSignature
	copy(ss2.Sig[:], ssb)
	t.Logf("ssb: %v ", ssb)
	ss2.Hint = '\u554a'
	t.Logf("ss2: %v ", ss2)
	t.Logf("ss2: %v ", ss2.String())

}
