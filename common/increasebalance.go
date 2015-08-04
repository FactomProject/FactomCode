// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
)

var IncreaseBalanceSize int = 32 + 4 + 32

type IncreaseBalance struct {
	Printable          `json:"-"`
	BinaryMarshallable `json:"-"`
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

	EncodeVarInt(buf, b.Index)

	EncodeVarInt(buf, b.NumEC)

	return buf.Bytes(), nil
}

func (b *IncreaseBalance) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	if err = b.readUnmarshal(buf); err != nil {
		return
	}
	newData = buf.Bytes()
	return
}

func (b *IncreaseBalance) UnmarshalBinary(data []byte) (err error) {
	_, err = b.UnmarshalBinaryData(data)
	return
}

func (b *IncreaseBalance) readUnmarshal(buf *bytes.Buffer) (err error) {
	tmp := make([]byte, 0)
	hash := make([]byte, 32)

	_, err = buf.Read(hash)
	if err != nil {
		return
	}
	b.ECPubKey = new([32]byte)
	copy(b.ECPubKey[:], hash)

	_, err = buf.Read(hash)
	if err != nil {
		return
	}
	if b.TXID == nil {
		b.TXID = NewHash()
	}
	b.TXID.SetBytes(hash)

	b.Index, tmp = DecodeVarInt(buf.Bytes())
	buf = bytes.NewBuffer(tmp)

	b.NumEC, tmp = DecodeVarInt(buf.Bytes())
	buf = bytes.NewBuffer(tmp)

	return
}

func (e *IncreaseBalance) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *IncreaseBalance) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *IncreaseBalance) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *IncreaseBalance) Spew() string {
	return Spew(e)
}
