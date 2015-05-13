package common_test

import (
	"fmt"
	"testing"
	
	"github.com/FactomProject/FactomCode/common"
)

func TestECBlockMarshal(t *testing.T) {
	fmt.Printf("TestECBlockMarshal\n---\n")
	ecb := common.NewECBlock()
	if p, err := ecb.MarshalBinary(); err != nil {
		t.Error(err)
	} else if z := make([]byte, 204); string(p) != string(z) {
		t.Errorf("Marshal failed on zeroed CommitChain")
	}
}