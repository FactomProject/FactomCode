package main 

import (
	"fmt"
	
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	
	"NotaryChain/notaryapi"
)

func main() {
	block := notaryapi.CreateBlock(nil, 1)
	
	entry := &notaryapi.MakeDataEntry()
	entry.UpdateData([]byte{0x10,0x11,0x12,0x13})
	
	key := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	
	entry.Sign(rand.Reader, key)
	
	block.AddEntry(entry)
}

