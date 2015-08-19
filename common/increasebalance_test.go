package common_test

import (
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)


func TestIncreaseBalanceMarshalUnmarshal(t *testing.T) {
	ib1 := NewIncreaseBalance()
	pub := new([32]byte)
	copy(pub[:], byteof(0xaa))
	ib1.ECPubKey = pub
	ib1.TXID.SetBytes(byteof(0xbb))
	ib1.NumEC = uint64(13)
	p, err := ib1.MarshalBinary()
	if err != nil {
		t.Error(err)
	}
	ib2 := NewIncreaseBalance()
	ib2.UnmarshalBinary(p)
	
	q, err := ib2.MarshalBinary()
	if err != nil {
		t.Error(err)
	}
	if string(p) != string(q) {
		t.Errorf("ib1 = %x\n", p)
		t.Errorf("ib2 = %x\n", q)
	}
}

func TestInvalidIncreaseBalanceUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestInvalidIncreaseBalanceUnmarshal\n---\n")

	ib := NewIncreaseBalance()
	_, err := ib.UnmarshalBinaryData(nil)
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	ib = NewIncreaseBalance()
	_, err = ib.UnmarshalBinaryData([]byte{})
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}

	ib = NewIncreaseBalance()
	err = ib.UnmarshalBinary(nil)
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	ib = NewIncreaseBalance()
	err = ib.UnmarshalBinary([]byte{})
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
}
