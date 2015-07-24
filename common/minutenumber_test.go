package common_test

import (
	"fmt"
	"testing"

	. "github.com/FactomProject/FactomCode/common"
)

func TestMinuteNumberMarshalUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestMinuteNumberMarshalUnmarshal\n---\n")

	mn := new(MinuteNumber)
	mn.Number = 5
	b, err := mn.MarshalBinary()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(b) != 1 {
		t.Error("Invalid byte length")
	}
	if b[0] != 5 {
		t.Error("Invalid byte")
	}
	mn2 := new(MinuteNumber)
	err = mn2.UnmarshalBinary(b)
	if err != nil {
		t.Error(err)
	}
	if mn2.Number != mn.Number {
		t.Error("Invalid data unmarshalled")
	}

	mn3 := new(MinuteNumber)
	remainder, err := mn3.UnmarshalBinaryData([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	if err != nil {
		t.Error(err)
	}
	if mn3.Number != 1 {
		t.Error("Invalid data unmarshalled")
	}
	if len(remainder) != 4 {
		t.Error("Invalid byte length")
	}
	if remainder[0] != 0x02 || remainder[1] != 0x03 || remainder[2] != 0x04 || remainder[3] != 0x05 {
		t.Error("Wrong remainder returned")
	}
}

func TestInvalidMinuteNumberUnmarshal(t *testing.T) {
	fmt.Printf("\n---\nTestInvalidMinuteNumberUnmarshal\n---\n")
	mn := new(MinuteNumber)
	_, err := mn.UnmarshalBinaryData(nil)
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	mn = new(MinuteNumber)
	_, err = mn.UnmarshalBinaryData([]byte{})
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}

	mn = new(MinuteNumber)
	err = mn.UnmarshalBinary(nil)
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
	mn = new(MinuteNumber)
	err = mn.UnmarshalBinary([]byte{})
	if err == nil {
		t.Error("We expected errors but we didn't get any")
	}
}
