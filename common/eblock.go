// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
)

const (
	EBHeaderSize = 140 // 32+32+32+32+4+4+4
)

// EBlock is the Entry Block. It holds the hashes of the Entries and its Merkel
// Root is written into the Directory Blocks. Each Entry Block represents all
// of the entries for a paticular Chain during a 10 minute period.
type EBlock struct {
	Header *EBlockHeader
	Body   *EBlockBody
}

var _ Printable = (*EBlock)(nil)
var _ BinaryMarshallable = (*EBlock)(nil)

func (c *EBlock) MarshalledSize() uint64 {
	return uint64(EBHeaderSize)
}

// MakeEBlock creates a new Entry Block belonging to the provieded Entry Chain.
// Its PrevKeyMR and PrevLedgerKeyMR are populated by the provided previous
// Entry Block. If The previous Entry Block is nil (the new Entry Block is
// first in the Chain) the relevent Entry Block Header fields will contain
// zeroed Hashes.
func MakeEBlock(echain *EChain, prev *EBlock) (*EBlock, error) {
	e := NewEBlock()
	e.Header.ChainID = echain.ChainID
	if prev != nil {
		var err error
		e.Header.PrevKeyMR, err = prev.KeyMR()
		if err != nil {
			return nil, err
		}
		e.Header.PrevLedgerKeyMR, err = prev.Hash()
		if err != nil {
			return nil, err
		}
	}
	e.Header.EBSequence = echain.NextBlockHeight
	return e, nil
}

// NewEBlock returns a blank initialized Entry Block with all of its fields
// zeroed.
func NewEBlock() *EBlock {
	e := new(EBlock)
	e.Header = NewEBlockHeader()
	e.Body = NewEBlockBody()
	return e
}

// AddEBEntry creates a new Entry Block Entry from the provided Factom Entry
// and adds it to the Entry Block Body.
func (e *EBlock) AddEBEntry(entry *Entry) error {
	e.Body.EBEntries = append(e.Body.EBEntries, entry.Hash())
	return nil
}

// AddEndOfMinuteMarker adds the End of Minute to the Entry Block. The End of
// Minut byte becomes the last byte in a 32 byte slice that is added to the
// Entry Block Body as an Entry Block Entry.
func (e *EBlock) AddEndOfMinuteMarker(m byte) {
	h := make([]byte, 32)
	h[len(h)-1] = m
	hash := NewHash()
	hash.SetBytes(h)
	e.Body.EBEntries = append(e.Body.EBEntries, hash)
}

// BuildHeader updates the Entry Block Header to include information about the
// Entry Block Body. BuildHeader should be run after the Entry Block Body has
// included all of its EntryEntries.
func (e *EBlock) BuildHeader() error {
	e.Header.BodyMR = e.Body.MR()
	e.Header.EntryCount = uint32(len(e.Body.EBEntries))
	return nil
}

// Hash returns the simple Sha256 hash of the serialized Entry Block. Hash is
// used to provide the PrevLedgerKeyMR to the next Entry Block in a Chain.
func (e *EBlock) Hash() (*Hash, error) {
	p, err := e.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return Sha(p), nil
}

// KeyMR returns the hash of the hash of the Entry Block Header concatinated
// with the Merkle Root of the Entry Block Body. The Body Merkle Root is
// calculated by the func (e *EBlockBody) MR() which is called by the func
// (e *EBlock) BuildHeader().
func (e *EBlock) KeyMR() (*Hash, error) {
	// Sha(Sha(header) + BodyMR)
	e.BuildHeader()
	header, err := e.marshalHeaderBinary()
	if err != nil {
		return nil, err
	}
	h := Sha(header)
	return Sha(append(h.Bytes(), e.Header.BodyMR.Bytes()...)), nil
}

// MarshalBinary returns the serialized binary form of the Entry Block.
func (e *EBlock) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := e.BuildHeader(); err != nil {
		return buf.Bytes(), err
	}
	if p, err := e.marshalHeaderBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	if p, err := e.marshalBodyBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary populates the Entry Block object from the serialized binary
// data.
func (e *EBlock) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	newData = data

	newData, err = e.unmarshalHeaderBinaryData(newData)
	if err != nil {
		return
	}

	newData, err = e.unmarshalBodyBinaryData(newData)
	if err != nil {
		return
	}

	return
}

func (e *EBlock) UnmarshalBinary(data []byte) (err error) {
	_, err = e.UnmarshalBinaryData(data)
	return
}

// marshalBodyBinary returns a serialized binary Entry Block Body
func (e *EBlock) marshalBodyBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, v := range e.Body.EBEntries {
		buf.Write(v.Bytes())
	}

	return buf.Bytes(), nil
}

// marshalHeaderBinary returns a serialized binary Entry Block Header
func (e *EBlock) marshalHeaderBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 32 byte ChainID
	buf.Write(e.Header.ChainID.Bytes())

	// 32 byte Body MR
	buf.Write(e.Header.BodyMR.Bytes())

	// 32 byte Previous Key MR
	buf.Write(e.Header.PrevKeyMR.Bytes())

	// 32 byte Previous Full Hash
	buf.Write(e.Header.PrevLedgerKeyMR.Bytes())

	if err := binary.Write(buf, binary.BigEndian, e.Header.EBSequence); err != nil {
		return buf.Bytes(), err
	}

	if err := binary.Write(buf, binary.BigEndian, e.Header.DBHeight); err != nil {
		return buf.Bytes(), err
	}

	if err := binary.Write(buf, binary.BigEndian, e.Header.EntryCount); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

// unmarshalBodyBinary builds the Entry Block Body from the serialized binary.
func (e *EBlock) unmarshalBodyBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	for i := uint32(0); i < e.Header.EntryCount; i++ {
		if _, err = buf.Read(hash); err != nil {
			return buf.Bytes(), err
		}

		h := NewHash()
		h.SetBytes(hash)
		e.Body.EBEntries = append(e.Body.EBEntries, h)
	}

	newData = buf.Bytes()
	return
}

func (e *EBlock) unmarshalBodyBinary(data []byte) (err error) {
	_, err = e.unmarshalBodyBinaryData(data)
	return
}

// unmarshalHeaderBinary builds the Entry Block Header from the serialized binary.
func (e *EBlock) unmarshalHeaderBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)
	newData = data

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.ChainID.SetBytes(hash)
	}

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.BodyMR.SetBytes(hash)
	}

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.PrevKeyMR.SetBytes(hash)
	}

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.PrevLedgerKeyMR.SetBytes(hash)
	}

	if err = binary.Read(buf, binary.BigEndian, &e.Header.EBSequence); err != nil {
		return
	}

	if err = binary.Read(buf, binary.BigEndian, &e.Header.DBHeight); err != nil {
		return
	}

	if err = binary.Read(buf, binary.BigEndian, &e.Header.EntryCount); err != nil {
		return
	}

	newData = buf.Bytes()

	return
}

func (e *EBlock) unmarshalHeaderBinary(data []byte) (err error) {
	_, err = e.unmarshalHeaderBinaryData(data)
	return
}

func (e *EBlock) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *EBlock) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *EBlock) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *EBlock) Spew() string {
	return Spew(e)
}

// EBlockBody is the series of Hashes that form the Entry Block Body.
type EBlockBody struct {
	EBEntries []*Hash
}

var _ Printable = (*EBlockBody)(nil)

// NewEBlockBody initalizes an empty Entry Block Body.
func NewEBlockBody() *EBlockBody {
	e := new(EBlockBody)
	e.EBEntries = make([]*Hash, 0)
	return e
}

// MR calculates the Merkle Root of the Entry Block Body. See func
// BuildMerkleTreeStore(hashes []*Hash) (merkles []*Hash) in common/merkle.go.
func (e *EBlockBody) MR() *Hash {
	mrs := BuildMerkleTreeStore(e.EBEntries)
	r := mrs[len(mrs)-1]
	return r
}

func (e *EBlockBody) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *EBlockBody) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *EBlockBody) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *EBlockBody) Spew() string {
	return Spew(e)
}

// EBlockHeader holds relevent metadata about the Entry Block and the data
// nessisary to verify the previous block in the Entry Block Chain.
type EBlockHeader struct {
	ChainID         *Hash
	BodyMR          *Hash
	PrevKeyMR       *Hash
	PrevLedgerKeyMR *Hash
	EBSequence      uint32
	DBHeight        uint32
	EntryCount      uint32
}

var _ Printable = (*EBlockHeader)(nil)

// NewEBlockHeader initializes a new empty Entry Block Header.
func NewEBlockHeader() *EBlockHeader {
	e := new(EBlockHeader)
	e.ChainID = NewHash()
	e.BodyMR = NewHash()
	e.PrevKeyMR = NewHash()
	e.PrevLedgerKeyMR = NewHash()
	return e
}

func (e *EBlockHeader) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *EBlockHeader) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *EBlockHeader) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *EBlockHeader) Spew() string {
	return Spew(e)
}
