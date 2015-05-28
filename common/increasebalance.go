// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
)

type IncreaseBalance struct {
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
	buf := new(bytes.Buffer)
	
	buf.Write(b.ECPubKey[:])
	if err := binary.Write(buf, binary.BigEndian, b.Credits); err != nil {
		return buf.Bytes(), err
	}
	buf.Write(b.FactomTxHash.Bytes)
	
	return buf.Bytes(), nil
}

func (b *IncreaseBalance) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	if _, err := buf.Read(b.ECPubKey[:]); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.Credits); err != nil {
		return err
	}
	if _, err := buf.Read(b.FactomTxHash.Bytes); err != nil {
		return err
	}
	
	return nil
}

