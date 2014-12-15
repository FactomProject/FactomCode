package main

import (
	"testing"
	"github.com/FactomProject/FactomCode/notaryapi"		
	"fmt"
	
)
func TestBuyCredit(t *testing.T) {
	
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	barray1 := (make([]byte, 32))
	barray1[31] = 2
	factoidTxHash := new (notaryapi.Hash)
	factoidTxHash.SetBytes(barray1)	
		
	_, err := processBuyEntryCredit(pubKey, 100, factoidTxHash)
	
	
	printCreditMap()
	
	printPaidEntryMap()
	printCChain()
				
	if err != nil {
		t.Errorf("Error:%v", err)
	}
} 

func TestAddChain(t *testing.T) {

	chain := new (notaryapi.EChain)
	bName := make ([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))	
	bName = append(bName, []byte("bookkeeping2"))		
	
	chain.Name = bName
	chain.GenerateIDFromName()
	
	entry := new (notaryapi.Entry)
	entry.ExtIDs = make ([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))	
	//entry.ExtIDs = append(entry.Extcd IDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))	
	//entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("First entry for chain:\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\"Rules:\"asl;djfasldkfjasldfjlksouiewopurw\"")
	
	chain.FirstEntry = entry
	
	binaryEntry, _ := entry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)
	
	entryChainIDHash := notaryapi.Sha(append(chain.ChainID.Bytes, entryHash.Bytes ...))
	
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	printCreditMap()
	printPaidEntryMap()
	printCChain()
	_, err := processCommitChain(entryHash, chain.ChainID, entryChainIDHash, pubKey)
	fmt.Println("after processCommitChain:")		
	printPaidEntryMap()

	if err != nil {
		fmt.Println("Error:%v", err)
	}
	
	// Reveal new chain
	_, err = processRevealChain(chain, pubKey)	
	fmt.Println("after processNewChain:")
	printPaidEntryMap()
	if err != nil {
		fmt.Println("Error:%v", err)
	}		

} 

func TestAddEntry(t *testing.T) {
	chain := new (notaryapi.EChain)
	bName := make ([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))	
	bName = append(bName, []byte("bookkeeping2"))		
	
	chain.Name = bName
	chain.GenerateIDFromName()
	

	
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	for i:=1; i<20; i++{
		
		entry := new (notaryapi.Entry)
		entry.ExtIDs = make ([][]byte, 0, 5)
		entry.ExtIDs = append(entry.ExtIDs, []byte(string(i)))	
		entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))	
		entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
		entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw\"")
		entry.ChainID = notaryapi.Hash{}
		entry.ChainID.Bytes = 	chain.ChainID.Bytes
	
		binaryEntry, _ := entry.MarshalBinary()
		entryHash := notaryapi.Sha(binaryEntry)		
		nounce := uint32(i)
			
		_, err := processCommitEntry(entryHash, pubKey, nounce)
		fmt.Println("after processCommitEntry:")			
		printCreditMap()
		printPaidEntryMap()
		printCChain()			
		if err != nil {
			t.Errorf("Error:%v", err)
		}
		
		// Reveal new entry
		processRevealEntry(entry, pubKey, nounce)
		fmt.Println("after processRevealEntry:")	
		printPaidEntryMap()	
		if err != nil {
			t.Errorf("Error:%v", err)
		}	
	}
	
} 
