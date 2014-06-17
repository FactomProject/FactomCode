package main 

import (
	"fmt"
	
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	
	"NotaryChain/notaryapi"
	
	"github.com/firelizzard18/gocoding"
)

func main() {
	block, err := notaryapi.CreateBlock(nil, 1)
	if err != nil { panic(err) }
	
	entry := notaryapi.MakeDataEntry()
	entry.UpdateData([]byte{0x10,0x11,0x12,0x13})
	
	_key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { panic(err) }
	key := &notaryapi.ECDSAPrivKey{_key}
	
	entry.Sign(rand.Reader, key)
	
	block.AddEntry(&entry)
	
	data, err := gocoding.Marshal(/*func (v interface{}) ([]byte, error) {
		return json.MarshalIndent(v, "\t", "")
	}, */block)
	if err != nil { panic(err) }
	
	fmt.Println(string(data))
	fmt.Println(block.Entries[0])
}

