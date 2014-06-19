package main 

import (
	"fmt"
	
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"io/ioutil"
	
	"NotaryChain/notaryapi"
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
	
	for i, block := range blocks {
		data, err := block.MarshalBinary()
		if err != nil { panic(err) }
		
		err = ioutil.WriteFile(fmt.Sprintf(`app/rest/store.%d.block`, i), data, 0777)
		if err != nil { panic(err) }
	}
}
