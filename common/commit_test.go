package common_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/FactomProject/FactomCode/common"
	ed "github.com/agl/ed25519"
)

var (
	_ = fmt.Sprint("testing")
)

func TestCommitEntryMarshal(t *testing.T) {
	fmt.Printf("TestCommitEntryMarshal\n---\n")
	rand, _ := os.Open("/dev/random")
	
	ce := common.NewCommitEntry()

	// test MarshalBinary on a zeroed CommitEntry
	if p, err := ce.MarshalBinary(); err != nil {
		t.Error(err)
	} else if z := make([]byte, 136); string(p) != string(z) {
		t.Errorf("Marshal failed on zeroed CommitEntry")
	}
	
	// build a CommitEntry for testing
	ce.Version = 0
	ce.MilliTime = &[6]byte{1, 1, 1, 1, 1, 1}
	ce.EntryHash.Bytes, _ = hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	ce.Credits = 1
	
	// make a key and sign the msg
	if pub, privkey, err := ed.GenerateKey(rand); err != nil {
		t.Error(err)
	} else {
		ce.ECPubKey = pub
		ce.Sig = ed.Sign(privkey, ce.CommitMsg())
	}
	
	// marshal and unmarshal the commit and see if it matches
	ce2 := common.NewCommitEntry()
	if p, err := ce.MarshalBinary(); err != nil {
		t.Error(err)
	} else {
		fmt.Printf("%x\n", p)
		ce2.UnmarshalBinary(p)
	}
	
	if !ce2.IsValid() {
		t.Errorf("signature did not match after unmarshalbinary")
	}
}

func TestCommitChainMarshal(t *testing.T) {
	fmt.Printf("TestCommitChainMarshal\n---\n")
	rand, _ := os.Open("/dev/random")
	
	cc := common.NewCommitChain()

	// test MarshalBinary on a zeroed CommitChain
	if p, err := cc.MarshalBinary(); err != nil {
		t.Error(err)
	} else if z := make([]byte, 200); string(p) != string(z) {
		t.Errorf("Marshal failed on zeroed CommitChain")
	}
	
	// build a CommitChain for testing
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
	
	// marshal and unmarshal the commit and see if it matches
	cc2 := common.NewCommitChain()
	if p, err := cc.MarshalBinary(); err != nil {
		t.Error(err)
	} else {
		fmt.Printf("%x\n", p)
		cc2.UnmarshalBinary(p)
	}
	
	if !cc2.IsValid() {
		t.Errorf("signature did not match after unmarshalbinary")
	}
}
