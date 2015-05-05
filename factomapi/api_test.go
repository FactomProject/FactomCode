package factomapi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
	
	"github.com/FactomProject/FactomCode/common"
)

var (
	_ = fmt.Sprint("testing")
)


func TestCommitEntry(t *testing.T) {
	testcommit, err := hex.DecodeString("0013db1b4f31b11af341223423eb25edc394ceeb6c5fb7f42f9f22640bc29b7a2e949f5dc685630153d2921f4a778f1c567a6bc367a925fc0ffb5af0a6e123ea772f63dbc2de99ba71b6bc773c70d362b922baff5a954c51502f7d5c818a131ed4c164b48071d98abe477155a9fd803e9dc536ada99dac14b58e1f2fdfe9ba10d2a14aed9a667c00")
	if err != nil {
		t.Error(err)
	}

	if err := NewCommitEntry(testcommit); err != nil {
		t.Error(err)
	}
}

func NewCommitEntry(d []byte) (err error) {
	var (
		ver    uint8
		mtime  = make([]byte, 6)
		ehash  = new(common.Hash)
		cred   uint8
		pubkey = new([32]byte)
		sig    = new([64]byte)
	)
	ehash.Bytes = make([]byte, 32)
		
	buf := bytes.NewBuffer(d)
	
	// 1 byte Version
	ver, err = buf.ReadByte()
	if err != nil {
		return err
	}
	fmt.Println("ver:", ver)
	
	// 6 byte milliTimestamp
	if _, err := buf.Read(mtime); err != nil {
		return err
	}
	fmt.Printf("mtime: %x\n", mtime)
	
	// 32 byte Entry Hash
	if _, err := buf.Read(ehash.Bytes); err != nil {
		return err
	}
	fmt.Println("ehash:", ehash)
	
	// 1 byte number of Entry Credits
	cred, err = buf.ReadByte()
	if err != nil {
		return err
	} else if cred > 10 {
		fmt.Errorf("Cannot commit Entry over 10 Entry Credits: got %d\n", cred)
	}
	fmt.Println("cred:", cred)
	
	// 32 byte Pubkey
	copy(pubkey[:], buf.Next(32))
	fmt.Printf("pubkey: %x\n", *pubkey)
	
	// 64 byte Signature
	copy(sig[:], buf.Next(64))
	fmt.Printf("sig: %x\n", *sig)	
	return nil
}
