package common

import (
	"bytes"
	"fmt"
	"testing"
)

func TestHash(t *testing.T) {
	fmt.Println("\nTest hash===========================================================================")

	h := new(Hash)
	h.Bytes = EC_CHAINID

	bytes1, err := h.MarshalBinary()
	if err != nil {
		t.Errorf("Error:%v", err)
	}	
	fmt.Printf("bytes1:%v\n", bytes1)

	h2 := new(Hash)
	err = h2.UnmarshalBinary(bytes1)
	fmt.Printf("h2.bytes:%v\n", h2.Bytes)	
	if err != nil {
		t.Errorf("Error:%v", err)
	}	

	bytes2, err := h2.MarshalBinary()
	if err != nil {
		t.Errorf("Error:%v", err)
	}	
	fmt.Printf("bytes2:%v\n", bytes2)

	if bytes.Compare(bytes1, bytes2) != 0 {
		t.Errorf("Invalid output")
	}

}
