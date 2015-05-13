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
	pub, privkey, err := ed.GenerateKey(rand)
	if err != nil {
		t.Error(err)
	}
	ce.ECPubKey = pub
	ce.Sig = ed.Sign(privkey, ce.CommitMsg())
	
	// marshal and unmarshal the commit and see if it matches
	c2 := common.NewCommitEntry()
	if p, err := ce.MarshalBinary(); err != nil {
		t.Error(err)
	} else {
		c2.UnmarshalBinary(p)
	}
	
	if !c2.IsValid() {
		t.Errorf("signature did not match after unmarshalbinary")
	}
}
