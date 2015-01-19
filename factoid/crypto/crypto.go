package crypto

import (
	"github.com/FactomProject/FactomCode/factoid/util"

	"github.com/obscuren/sha3"
)

func Sha3Bin(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

// Creates an address given the bytes and the nonce
func CreateAddress(b []byte, nonce uint64) []byte {
	return Sha3Bin(util.NewValue([]interface{}{b, nonce}).Encode())[12:]
}
