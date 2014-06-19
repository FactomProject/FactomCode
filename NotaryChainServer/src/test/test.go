package main 

import (
	"os"
	
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	
	"NotaryChain/notaryapi"
	
	"github.com/firelizzard18/gocoding/json"
)

func main() {
	block, err := notaryapi.CreateBlock(nil, 1)
	if err != nil { panic(err) }
	
	entry := notaryapi.NewDataEntry([]byte{0x10,0x11,0x12,0x13})
	
	_key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { panic(err) }
	key := &notaryapi.ECDSAPrivKey{_key}
	
	entry.Sign(rand.Reader, key)
	
	block.AddEntry(entry)
	
	marshaller := json.NewMarshaller(os.Stdout)
	
	err = marshaller.Marshal(block)
	if err != nil { panic(err) }
}
