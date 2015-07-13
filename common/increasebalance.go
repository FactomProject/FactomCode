// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"fmt"
)

var _ = fmt.Println

var IncreaseBalanceSize int = 32 + 4 + 32

type IncreaseBalance struct {
	ECBlockEntry
	ECPubKey *[32]byte
	TXID     *Hash
	Index    uint64
	NumEC    uint64
}

func MakeIncreaseBalance(pubkey *[32]byte, facTX *Hash, credits int32) *IncreaseBalance {
	b := new(IncreaseBalance)
	b.ECPubKey = pubkey
	b.TXID = facTX
	b.NumEC = uint64(credits)
	return b
}

func NewIncreaseBalance() *IncreaseBalance {
	r := new(IncreaseBalance)
	r.TXID = NewHash()
	return r
}

func (b *IncreaseBalance) ECID() byte {
	return ECIDBalanceIncrease
}

func (b *IncreaseBalance) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(b.ECPubKey[:])
	
	buf.Write(b.TXID.Bytes())

	WriteVarInt(buf, b.Index)
	
	WriteVarInt(buf, b.NumEC)
	
	return buf.Bytes(), nil
}

func (b *IncreaseBalance) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := b.readUnmarshal(buf); err != nil {
		return err
	}
	return nil
}

func (b *IncreaseBalance) readUnmarshal(buf *bytes.Buffer) error {
	hash := make([]byte, 32)

	buf.Read(hash)
	b.ECPubKey = new([32]byte)
	copy(b.ECPubKey[:], hash)

	buf.Read(hash)
	b.TXID.SetBytes(hash)
	
	b.Index = ReadVarInt(buf)

	b.NumEC = ReadVarInt(buf)
	
	return nil
}