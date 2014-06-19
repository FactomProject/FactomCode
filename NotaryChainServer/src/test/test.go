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
	var err error
	
	blocks := make([]*notaryapi.Block, 2)
	
	blocks[0], err = notaryapi.CreateBlock(nil, 0)
	if err != nil { panic(err) }
	
	blocks[1], err = notaryapi.CreateBlock(blocks[0], 1)
	if err != nil { panic(err) }
	
	entry := notaryapi.NewDataEntry([]byte{0x10,0x11,0x12,0x13})
	
	_key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { panic(err) }
	key := &notaryapi.ECDSAPrivKey{_key}
	
	entry.Sign(rand.Reader, key)
	
	blocks[1].AddEntry(entry)
	
	marshaller := json.NewMarshaller(os.Stdout)
	
	err = marshaller.Marshal(blocks)
	if err != nil { panic(err) }
}
