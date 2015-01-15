package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"bufio"
	"testing"
	"github.com/FactomProject/FactomCode/notaryapi"		
	"time"
	
)

func UnmarshalJSON(b []byte) (*notaryapi.Entry, error) {
	type entry struct {
		ChainID string
		ExtIDs  []string
		Data    string
	}
	
	var je entry
	e := new(notaryapi.Entry)
	
	err := json.Unmarshal(b, &je)
	if err != nil {
		return nil, err
	}
	
	bytes, err := hex.DecodeString(je.ChainID)
	if err != nil {
		return nil, err
	}
	e.ChainID.Bytes = bytes
	
	for _, v := range je.ExtIDs {
		e.ExtIDs = append(e.ExtIDs, []byte(v))
	}
	bytes, err = hex.DecodeString(je.Data)
	if err != nil {
		return nil, err
	}
	e.Data = bytes
	
	return e, nil
}

func TestBuyCredit(t *testing.T) {
	
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	barray1 := (make([]byte, 32))
	barray1[31] = 2
	factoidTxHash := new (notaryapi.Hash)
	factoidTxHash.SetBytes(barray1)	
		
	_, err := processBuyEntryCredit(pubKey, 200000, factoidTxHash)
	
	
	printCreditMap()
	
	printPaidEntryMap()
	printCChain()
				
	if err != nil {
		t.Errorf("Error:%v", err)
	}
} 

func TestAddEntry(t *testing.T) {
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	file, _ := os.Open("/tmp/gutenberg/entries")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		e, err := UnmarshalJSON([]byte(scanner.Text()))
		if err != nil {
			t.Errorf("Error:", err)
		}

		binaryEntry, _ := e.MarshalBinary()
		entryHash := notaryapi.Sha(binaryEntry)		
		timestamp := time.Now().Unix()
		processCommitEntry(entryHash, pubKey, timestamp)
		time.Sleep(time.Second / 500)
		processRevealEntry(e)		
	}	
} 
