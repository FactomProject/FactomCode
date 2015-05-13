package common_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	
	"github.com/FactomProject/FactomCode/common"
	ed "github.com/agl/ed25519"
)

func TestECBlockMarshal(t *testing.T) {
	fmt.Printf("TestECBlockMarshal\n---\n")
	ecb := common.NewECBlock()
	if p, err := ecb.MarshalBinary(); err != nil {
		t.Error(err)
	} else if z := make([]byte, 204); string(p) != string(z) {
		t.Errorf("Marshal failed on zeroed CommitChain")
	}
	
	// build a CommitChain for testing
	rand, _ := os.Open("/dev/random")
	cc := common.NewCommitChain()
	cc.Version = 0
	cc.MilliTime = &[6]byte{1, 1, 1, 1, 1, 1}
	cc.ChainIDHash.Bytes, _ = hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	cc.Weld.Bytes, _ = hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	cc.EntryHash.Bytes, _ = hex.DecodeString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	cc.Credits = 11
	
	// make a key and sign the msg
	if pub, privkey, err := ed.GenerateKey(rand); err != nil {
		t.Error(err)
	} else {
		cc.ECPubKey = pub
		cc.Sig = ed.Sign(privkey, cc.CommitMsg())
	}

	// create a ECBlock for testing
	ecb.Header.ECChainID.Bytes, _ = hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")
	ecb.Header.BodyHash.Bytes, _ = hex.DecodeString("2222222222222222222222222222222222222222222222222222222222222222")
	ecb.Header.PrevKeyMR.Bytes, _ = hex.DecodeString("3333333333333333333333333333333333333333333333333333333333333333")
	ecb.Header.PrevHash3.Bytes, _ = hex.DecodeString("4444444444444444444444444444444444444444444444444444444444444444")
	ecb.Header.DBHeight = 10
	ecb.Header.SegmentsMR.Bytes, _ = hex.DecodeString("5555555555555555555555555555555555555555555555555555555555555555")
	ecb.Header.BalanceCommit.Bytes, _ = hex.DecodeString("6666666666666666666666666666666666666666666666666666666666666666")
	ecb.Header.ObjectCount = 0
	
	// add the CommitChain to the ECBlock
	ecb.AddEntry(cc)
	
	ecb2 := common.NewECBlock()
	if p, err := ecb.MarshalBinary(); err != nil {
		t.Error(err)
	} else {
		fmt.Printf("%x\n", p)
		if err := ecb2.UnmarshalBinary(p); err != nil {
			t.Error(err)
		}
		if q, err := ecb2.MarshalBinary(); err != nil {
			t.Error(err)
		} else if string(p) != string(q) {
			t.Errorf("ecb = %x\necb2 = %x\n", p, q)
		}
	}
		
}