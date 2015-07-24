package common_test

import (
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)

/*
func TestIncreaseBalanceMarshalUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestIncreaseBalanceMarshalUnmarshal\n---\n")

}*/

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
