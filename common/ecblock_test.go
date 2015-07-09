package common_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	
	"github.com/davecgh/go-spew/spew"
	"github.com/FactomProject/FactomCode/common"
	ed "github.com/FactomProject/ed25519"
)

func TestECBlockMarshal(t *testing.T) {
	fmt.Printf("---\nTestECBlockMarshal\n---\n")
	ecb := common.NewECBlock()
	
	// build a CommitChain for testing
	rand, _ := os.Open("/dev/random")
	cc := common.NewCommitChain()
	cc.Version = 0
	cc.MilliTime = &[6]byte{1, 1, 1, 1, 1, 1}
	p, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	cc.ChainIDHash.SetBytes(p)
	p, _ = hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	cc.Weld.SetBytes(p)
	p, _ = hex.DecodeString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	cc.EntryHash.SetBytes(p)
	cc.Credits = 11
	
	// make a key and sign the msg
	if pub, privkey, err := ed.GenerateKey(rand); err != nil {
		t.Error(err)
	} else {
		cc.ECPubKey = pub
		cc.Sig = ed.Sign(privkey, cc.CommitMsg())
	}
	
	// create an IncreaseBalance for testing
	pub := new([32]byte)
	p, _ = hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	copy(pub[:], p)
	facTX := common.NewHash()
	p, _ = hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	facTX.SetBytes(p)
	ib := common.NewIncreaseBalance(pub, facTX, 12)

	// create a ECBlock for testing
	p, _ = hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")
	ecb.Header.ECChainID.SetBytes(p)
	p, _ = hex.DecodeString("2222222222222222222222222222222222222222222222222222222222222222")
	ecb.Header.BodyHash.SetBytes(p)
	p, _ = hex.DecodeString("3333333333333333333333333333333333333333333333333333333333333333")
	ecb.Header.PrevHeaderHash.SetBytes(p)
	p, _ = hex.DecodeString("4444444444444444444444444444444444444444444444444444444444444444")
	ecb.Header.PrevFullHash.SetBytes(p)
	ecb.Header.DBHeight = 10
	ecb.Header.HeaderExpansionArea, _ = hex.DecodeString("5555555555555555555555555555555555555555555555555555555555555555")
	p, _ = hex.DecodeString("6666666666666666666666666666666666666666666666666666666666666666")
	ecb.Header.ObjectCount = 0
	
	// add the CommitChain to the ECBlock
	ecb.AddEntry(cc)
	
	// add the IncreaseBalance
	ecb.AddEntry(ib)
	
	// add the MinuteNumber
	min := common.NewMinuteNumber()
	min.Number = 3
	ecb.AddEntry(min)
	
	fmt.Println(spew.Sdump(ecb))
	
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
		fmt.Println(spew.Sdump(ecb2))
	}
}
