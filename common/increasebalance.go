// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
    "fmt"
)
var _ = fmt.Println

var IncreaseBalanceSize int = 32+4+32
type IncreaseBalance struct {
    ECBlockEntry
	ECPubKey *[32]byte
	Credits int32
	FactomTxHash *Hash
}

func NewIncreaseBalance(pubkey *[32]byte, facTX *Hash, credits int32) *IncreaseBalance {
	b := new(IncreaseBalance)
	b.ECPubKey = pubkey
	b.Credits = credits
	b.FactomTxHash = facTX
	return b
}

func (b *IncreaseBalance) ECID() byte {
	return ECIDBalanceIncrease
}

func (b *IncreaseBalance) MarshalBinary() ([]byte, error) {
    fmt.Println("Marshalling IB: ", b.Credits)
	buf := new(bytes.Buffer)
	
	buf.Write(b.ECPubKey[:])
	
    err := binary.Write(buf, binary.BigEndian, b.Credits); 
    if err != nil { return buf.Bytes(), err }
    
	buf.Write(b.FactomTxHash.Bytes())
	
	return buf.Bytes(), nil
}

func (b *IncreaseBalance) UnmarshalBinary(data []byte) error {
    
    b.ECPubKey = new([32]byte)
    copy(b.ECPubKey[:],data[:32]); data = data[32:]
    
    b.Credits, data = int32(binary.BigEndian.Uint32(data)),data[4:]
    
    b.FactomTxHash = new(Hash)
    b.FactomTxHash.SetBytes(data[:32])
    
    return nil
    
}

