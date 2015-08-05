// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	ECIDServerIndexNumber byte = iota
	ECIDMinuteNumber
	ECIDChainCommit
	ECIDEntryCommit
	ECIDBalanceIncrease
)

// The Entry Credit Block consists of a header and a body. The body is composed
// of primarily Commits and Balance Increases with Minute Markers and Server
// Markers distributed throughout.
type ECBlock struct {
	Header *ECBlockHeader
	Body   *ECBlockBody
}

var _ Printable = (*ECBlock)(nil)
var _ BinaryMarshallable = (*ECBlock)(nil)

func (c *ECBlock) MarshalledSize() uint64 {
	panic("Function not implemented")
	return 0
}

func NewECBlock() *ECBlock {
	e := new(ECBlock)
	e.Header = NewECBlockHeader()
	e.Body = NewECBlockBody()
	return e
}

func NextECBlock(prev *ECBlock) *ECBlock {
	e := NewECBlock()
	e.Header.PrevHeaderHash = prev.HeaderHash()
	e.Header.PrevLedgerKeyMR = prev.Hash()
	e.Header.DBHeight = prev.Header.DBHeight + 1
	return e
}

func (e *ECBlock) AddEntry(entries ...ECBlockEntry) {
	e.Body.Entries = append(e.Body.Entries, entries...)
}

func (e *ECBlock) Hash() *Hash {
	p, err := e.MarshalBinary()
	if err != nil {
		return NewHash()
	}
	return Sha(p)
}

func (e *ECBlock) HeaderHash() *Hash {
	p, err := e.marshalHeaderBinary()
	if err != nil {
		return NewHash()
	}
	return Sha(p)
}

func (e *ECBlock) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Header
	if err := e.BuildHeader(); err != nil {
		return buf.Bytes(), err
	}
	if p, err := e.marshalHeaderBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	// Body of ECBlockEntries
	if p, err := e.marshalBodyBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func (e *ECBlock) BuildHeader() error {
	// Marshal the Body
	p, err := e.marshalBodyBinary()
	if err != nil {
		return err
	}

	e.Header.BodyHash = Sha(p)
	e.Header.ObjectCount = uint64(len(e.Body.Entries))
	e.Header.BodySize = uint64(len(p))

	return nil
}

func (e *ECBlock) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	// Unmarshal Header
	newData, err = e.unmarshalHeaderBinaryData(data)
	if err != nil {
		return
	}

	// Unmarshal Body
	newData, err = e.unmarshalBodyBinaryData(newData)
	if err != nil {
		return
	}

	return
}

func (e *ECBlock) UnmarshalBinary(data []byte) (err error) {
	_, err = e.UnmarshalBinaryData(data)
	return
}

func (e *ECBlock) marshalBodyBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, v := range e.Body.Entries {
		p, err := v.MarshalBinary()
		if err != nil {
			return buf.Bytes(), err
		}
		buf.WriteByte(v.ECID())
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func (e *ECBlock) marshalHeaderBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 32 byte ECChainID
	buf.Write(e.Header.ECChainID.Bytes())

	// 32 byte BodyHash
	buf.Write(e.Header.BodyHash.Bytes())

	// 32 byte Previous Header Hash
	buf.Write(e.Header.PrevHeaderHash.Bytes())

	// 32 byte Previous Full Hash
	buf.Write(e.Header.PrevLedgerKeyMR.Bytes())

	// 4 byte Directory Block Height
	if err := binary.Write(buf, binary.BigEndian, e.Header.DBHeight); err != nil {
		return buf.Bytes(), err
	}

	// variable Header Expansion Size
	if err := EncodeVarInt(buf,
		uint64(len(e.Header.HeaderExpansionArea))); err != nil {
		return buf.Bytes(), err
	}

	// varable byte Header Expansion Area
	buf.Write(e.Header.HeaderExpansionArea)

	// 8 byte Object Count
	if err := binary.Write(buf, binary.BigEndian, e.Header.ObjectCount); err != nil {
		return buf.Bytes(), err
	}

	// 8 byte size of the Body
	if err := binary.Write(buf, binary.BigEndian, e.Header.BodySize); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

func (e *ECBlock) unmarshalBodyBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)

	for i := uint64(0); i < e.Header.ObjectCount; i++ {
		var id byte
		id, err = buf.ReadByte()
		if err != nil {
			newData = buf.Bytes()
			return
		}
		switch id {
		case ECIDServerIndexNumber:
			s := NewServerIndexNumber()
			if buf.Len() < ServerIndexNumberSize {
				err = io.EOF
				newData = buf.Bytes()
				return
			}
			_, err = s.UnmarshalBinaryData(buf.Next(ServerIndexNumberSize))
			if err != nil {
				newData = buf.Bytes()
				return
			}
			e.Body.Entries = append(e.Body.Entries, s)
		case ECIDMinuteNumber:
			m := NewMinuteNumber()
			if buf.Len() < MinuteNumberSize {
				err = io.EOF
				newData = buf.Bytes()
				return
			}
			_, err = m.UnmarshalBinaryData(buf.Next(MinuteNumberSize))
			if err != nil {
				newData = buf.Bytes()
				return
			}
			e.Body.Entries = append(e.Body.Entries, m)
		case ECIDChainCommit:
			if buf.Len() < CommitChainSize {
				err = io.EOF
				newData = buf.Bytes()
				return
			}
			c := NewCommitChain()
			_, err = c.UnmarshalBinaryData(buf.Next(CommitChainSize))
			if err != nil {
				return
			}
			e.Body.Entries = append(e.Body.Entries, c)
		case ECIDEntryCommit:
			if buf.Len() < CommitEntrySize {
				err = io.EOF
				newData = buf.Bytes()
				return
			}
			c := NewCommitEntry()
			_, err = c.UnmarshalBinaryData(buf.Next(CommitEntrySize))
			if err != nil {
				return
			}
			e.Body.Entries = append(e.Body.Entries, c)
		case ECIDBalanceIncrease:
			c := NewIncreaseBalance()
			err = c.readUnmarshal(buf)
			if err != nil {
				return
			}
			e.Body.Entries = append(e.Body.Entries, c)
		default:
			err = fmt.Errorf("Unsupported ECID: %x\n", id)
			return
		}
	}

	newData = buf.Bytes()
	return
}

func (b *ECBlock) unmarshalBodyBinary(data []byte) (err error) {
	_, err = b.unmarshalBodyBinaryData(data)
	return
}

func (e *ECBlock) unmarshalHeaderBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.ECChainID.SetBytes(hash)
	}

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.BodyHash.SetBytes(hash)
	}

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.PrevHeaderHash.SetBytes(hash)
	}

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.Header.PrevLedgerKeyMR.SetBytes(hash)
	}

	if err = binary.Read(buf, binary.BigEndian, &e.Header.DBHeight); err != nil {
		return
	}

	// read the Header Expansion Area
	hesize, tmp := DecodeVarInt(buf.Bytes())
	buf = bytes.NewBuffer(tmp)
	e.Header.HeaderExpansionArea = make([]byte, hesize)
	if _, err = buf.Read(e.Header.HeaderExpansionArea); err != nil {
		return
	}

	if err = binary.Read(buf, binary.BigEndian, &e.Header.ObjectCount); err != nil {
		return
	}

	if err = binary.Read(buf, binary.BigEndian, &e.Header.BodySize); err != nil {
		return
	}

	newData = buf.Bytes()
	return
}

func (e *ECBlock) unmarshalHeaderBinary(data []byte) error {
	_, err := e.unmarshalHeaderBinaryData(data)
	return err
}

func (e *ECBlock) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ECBlock) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ECBlock) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ECBlock) Spew() string {
	return Spew(e)
}

type ECBlockBody struct {
	Entries []ECBlockEntry
}

var _ Printable = (*ECBlockBody)(nil)

func NewECBlockBody() *ECBlockBody {
	b := new(ECBlockBody)
	b.Entries = make([]ECBlockEntry, 0)
	return b
}

func (e *ECBlockBody) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ECBlockBody) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ECBlockBody) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ECBlockBody) Spew() string {
	return Spew(e)
}

type ECBlockEntry interface {
	Printable

	ECID() byte
	MarshalBinary() ([]byte, error)
	UnmarshalBinary(data []byte) error
}

type ECBlockHeader struct {
	ECChainID           *Hash
	BodyHash            *Hash
	PrevHeaderHash      *Hash
	PrevLedgerKeyMR        *Hash
	DBHeight            uint32
	HeaderExpansionArea []byte
	ObjectCount         uint64
	BodySize            uint64
}

var _ Printable = (*ECBlockHeader)(nil)

func NewECBlockHeader() *ECBlockHeader {
	h := new(ECBlockHeader)
	h.ECChainID = NewHash()
	h.ECChainID.SetBytes(EC_CHAINID)
	h.BodyHash = NewHash()
	h.PrevHeaderHash = NewHash()
	h.PrevLedgerKeyMR = NewHash()
	h.HeaderExpansionArea = make([]byte, 0)
	return h
}

func (e *ECBlockHeader) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ECBlockHeader) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ECBlockHeader) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ECBlockHeader) Spew() string {
	return Spew(e)
}
