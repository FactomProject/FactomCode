package main

import (
	"fmt"
	"testing"
	"github.com/FactomProject/FactomCode/notaryapi"		
	"encoding/hex"	
)
func TestBuyCredit(t *testing.T) {
	barray, err := hex.DecodeString("dd34357d8147e8a177811f1779f394e1c0f88a09d52ed0c9c2f4d4793fdd9fb05781167c351a2fba45444426890a1793c9327c8d01d45a47df4851a333d0e80d")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(barray)
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	barray1 := (make([]byte, 32))
	barray1[31] = 2
	factoidTxHash := new (notaryapi.Hash)
	factoidTxHash.SetBytes(barray1)	
		
	_, err = processBuyEntryCredit(pubKey, 200000, factoidTxHash)
	
	
	printCreditMap()
	
	printPaidEntryMap()
	printCChain()
				
	if err != nil {
		t.Errorf("Error:%v", err)
	}
}
