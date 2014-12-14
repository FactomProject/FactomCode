package main

import (
	"testing"
	"github.com/FactomProject/FactomCode/notaryapi"		
	"github.com/FactomProject/FactomCode/factomapi"		
	"fmt"	
	"bytes"
	"strings"
	"reflect"
	
)
func TestBuyCredit(t *testing.T) {
	
	barray := (make([]byte, 32))
	barray[0] = 1
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	barray1 := (make([]byte, 32))
	barray1[31] = 2
	factoidTxHash := new (notaryapi.Hash)
	factoidTxHash.SetBytes(barray1)	
		
	_, err := processBuyEntryCredit(pubKey, 20, factoidTxHash)
	
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
	bName = append(bName, []byte("bookkeeping"))		
	
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
	
	binaryChainID, _ := chain.ChainID.MarshalBinary()
	chainIDHash := notaryapi.Sha(binaryChainID)
	
	entryChainIDHash := notaryapi.Sha(append(chain.ChainID.Bytes, entryHash.Bytes ...))
	
	barray := (make([]byte, 32))
	barray[0] = 1
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	printCreditMap()
	
	printPaidEntryMap()
	printCChain()
	_, err := processCommitChain(entryHash, chainIDHash, entryChainIDHash, pubKey)

		
	if err != nil {
		t.Errorf("Error:%v", err)
	}
} 

func TestAddEntry(t *testing.T) {
	chain := new (notaryapi.EChain)
	bName := make ([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))	
	bName = append(bName, []byte("bookkeeping"))		
	
	chain.Name = bName
	chain.GenerateIDFromName()
	
	entry := new (notaryapi.Entry)
	entry.ExtIDs = make ([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))	
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))	
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw\"")
	entry.ChainID = *chain.ChainID

	binaryEntry, _ := entry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)
	
	barray := (make([]byte, 32))
	barray[0] = 1
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)	
	
	nounce := uint32(1)
		
	_, err := processCommitEntry(entryHash, pubKey, nounce)
	printCreditMap()
	
	printPaidEntryMap()
	printCChain()			
	if err != nil {
		t.Errorf("Error:%v", err)
	}
} 

func printCreditMap(){
	
	fmt.Println("eCreditMap:")
	for key := range eCreditMap {
		fmt.Println("Key:", key, "Value", eCreditMap[key])
	}
}

func printPaidEntryMap(){
	
	fmt.Println("prePaidEntryMap:")
	for key := range prePaidEntryMap {
		fmt.Println("Key:", key, "Value", prePaidEntryMap[key])
	}
}

func printCChain(){
	
	fmt.Println("cchain:", cchain.ChainID.String())
	
	for i, block := range cchain.Blocks {
		if !block.IsSealed{
			continue
		}
		var buf bytes.Buffer
		err := factomapi.SafeMarshal(&buf, block.Header)
		
		fmt.Println("block.Header", string(i), ":", string(buf.Bytes()))	
		
		for _, cbentry := range block.CBEntries {
			t := reflect.TypeOf(cbentry)
			fmt.Println("cbEntry Type:", t.Name(), t.String())
			if strings.Contains(t.String(), "PayChainCBEntry"){
				fmt.Println("PayChainCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err!=nil{
					fmt.Println("Error:%v", err)
				}
				
				fmt.Println("PayChainCBEntry JSON",  ":", string(buf.Bytes()))			
						
			} else 	if strings.Contains(t.String(), "PayEntryCBEntry"){
				fmt.Println("PayEntryCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err!=nil{
					fmt.Println("Error:%v", err)
				}
				
				fmt.Println("PayEntryCBEntry JSON",  ":", string(buf.Bytes()))					

			} else 	if strings.Contains(t.String(), "BuyCBEntry"){
				fmt.Println("BuyCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err!=nil{
					fmt.Println("Error:%v", err)
				}			
				fmt.Println("BuyCBEntry JSON",  ":", string(buf.Bytes()))					
			}			
		}
		
		if err != nil {
	
			fmt.Println("Error:%v", err)
		}
	}

}
