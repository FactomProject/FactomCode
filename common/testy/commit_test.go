package common_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/FactomProject/FactomCode/common"
	ed "github.com/FactomProject/ed25519"
)

var (
	_ = fmt.Sprint("testing")
)

func TestCommitEntryMarshal(t *testing.T) {
	fmt.Printf("---\nTestCommitEntryMarshal\n---\n")

	ce := common.NewCommitEntry()

	// test MarshalBinary on a zeroed CommitEntry
	if p, err := ce.MarshalBinary(); err != nil {
		t.Error(err)
	} else if z := make([]byte, common.CommitEntrySize); string(p) != string(z) {
		t.Errorf("Marshal failed on zeroed CommitEntry")
	}

	// build a CommitEntry for testing
	ce.Version = 0
	ce.MilliTime = (*common.ByteSlice6)(&[6]byte{1, 1, 1, 1, 1, 1})
	p, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	ce.EntryHash.SetBytes(p)
	ce.Credits = 1

	// make a key and sign the msg
	if pub, privkey, err := ed.GenerateKey(rand.Reader); err != nil {
		t.Error(err)
	} else {
		ce.ECPubKey = (*common.ByteSlice32)(pub)
		ce.Sig = (*common.ByteSlice64)(ed.Sign(privkey, ce.CommitMsg()))
	}

	// marshal and unmarshal the commit and see if it matches
	ce2 := common.NewCommitEntry()
	if p, err := ce.MarshalBinary(); err != nil {
		t.Error(err)
	} else {
		t.Logf("%x\n", p)
		ce2.UnmarshalBinary(p)
	}

	if !ce2.IsValid() {
		t.Errorf("signature did not match after unmarshalbinary")
	}
}

func TestCommitChainMarshal(t *testing.T) {
	fmt.Printf("---\nTestCommitChainMarshal\n---\n")

	cc := common.NewCommitChain()

	// test MarshalBinary on a zeroed CommitChain
	if p, err := cc.MarshalBinary(); err != nil {
		t.Error(err)
	} else if z := make([]byte, common.CommitChainSize); string(p) != string(z) {
		t.Errorf("Marshal failed on zeroed CommitChain")
	}

	// build a CommitChain for testing
	cc.Version = 0
	cc.MilliTime = (*common.ByteSlice6)(&[6]byte{1, 1, 1, 1, 1, 1})
	p, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	cc.ChainIDHash.SetBytes(p)
	p, _ = hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	cc.Weld.SetBytes(p)
	p, _ = hex.DecodeString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	cc.EntryHash.SetBytes(p)
	cc.Credits = 11

	// make a key and sign the msg
	if pub, privkey, err := ed.GenerateKey(rand.Reader); err != nil {
		t.Error(err)
	} else {
		cc.ECPubKey = (*common.ByteSlice32)(pub)
		cc.Sig = (*common.ByteSlice64)(ed.Sign(privkey, cc.CommitMsg()))
	}

	// marshal and unmarshal the commit and see if it matches
	cc2 := common.NewCommitChain()
	if p, err := cc.MarshalBinary(); err != nil {
		t.Error(err)
	} else {
		t.Logf("%x\n", p)
		cc2.UnmarshalBinary(p)
	}

	if !cc2.IsValid() {
		t.Errorf("signature did not match after unmarshalbinary")
	}
}
