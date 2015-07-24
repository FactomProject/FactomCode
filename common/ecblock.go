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

func NewECBlock() *ECBlock {
	e := new(ECBlock)
	e.Header = NewECBlockHeader()
	e.Body = NewECBlockBody()
	return e
}

func NextECBlock(prev *ECBlock) *ECBlock {
	e := NewECBlock()
	e.Header.PrevHeaderHash = prev.Header.Hash()
	e.Header.PrevFullHash = prev.Hash()
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

func (e *ECBlock) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Header
	if err := e.BuildHeader(); err != nil {
		return buf.Bytes(), err
	}
	if p, err := e.Header.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	// Body of ECBlockEntries
	if p, err := e.Body.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func (e *ECBlock) BuildHeader() error {
	// Marshal the Body
	p, err := e.Body.MarshalBinary()
	if err != nil {
		return err
	}

	e.Header.BodyHash = Sha(p)
	e.Header.ObjectCount = uint64(len(e.Body.Entries))
	e.Header.BodySize = uint64(len(p))

	return nil
}

func (e *ECBlock) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)

	// Unmarshal Header
	e.Header.readUnmarshal(buf)

	// Unmarshal Body
	if err = e.Body.UnmarshalBinary(buf.Bytes()); err != nil {
		return
	}

	return
}

func (e *ECBlock) UnmarshalBinary(data []byte) (err error) {
	_, err = e.UnmarshalBinaryData(data)
	return
}

type ECBlockBody struct {
	Entries []ECBlockEntry
}

func NewECBlockBody() *ECBlockBody {
	b := new(ECBlockBody)
	b.Entries = make([]ECBlockEntry, 0)
	return b
}

func (b *ECBlockBody) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, v := range b.Entries {
		p, err := v.MarshalBinary()
		if err != nil {
			return buf.Bytes(), err
		}
		buf.WriteByte(v.ECID())
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func (b *ECBlockBody) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	for {
		id, err := buf.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch id {
		case ECIDServerIndexNumber:
		case ECIDMinuteNumber:
			m := NewMinuteNumber()
			err := m.UnmarshalBinary(buf.Next(MinuteNumberSize))
			if err != nil {
				return err
			}
			b.Entries = append(b.Entries, m)
		case ECIDChainCommit:
			c := NewCommitChain()
			err := c.UnmarshalBinary(buf.Next(CommitChainSize))
			if err != nil {
				return err
			}
			b.Entries = append(b.Entries, c)
		case ECIDEntryCommit:
			c := NewCommitEntry()
			err := c.UnmarshalBinary(buf.Next(CommitEntrySize))
			if err != nil {
				return err
			}
			b.Entries = append(b.Entries, c)
		case ECIDBalanceIncrease:
			c := NewIncreaseBalance()
			err := c.readUnmarshal(buf)
			if err != nil {
				return err
			}
			b.Entries = append(b.Entries, c)
		default:
			return fmt.Errorf("Unsupported ECID: %x\n", id)
		}
	}

	return nil
}

type ECBlockEntry interface {
	ECID() byte
	MarshalBinary() ([]byte, error)
	UnmarshalBinary(data []byte) error
}

type ECBlockHeader struct {
	ECChainID           *Hash
	BodyHash            *Hash
	PrevHeaderHash      *Hash
	PrevFullHash        *Hash
	DBHeight            uint32
	HeaderExpansionArea []byte
	ObjectCount         uint64
	BodySize            uint64
}

func NewECBlockHeader() *ECBlockHeader {
	h := new(ECBlockHeader)
	h.ECChainID = NewHash()
	h.ECChainID.SetBytes(EC_CHAINID)
	h.BodyHash = NewHash()
	h.PrevHeaderHash = NewHash()
	h.PrevFullHash = NewHash()
	h.HeaderExpansionArea = make([]byte, 0)
	return h
}

func (e *ECBlockHeader) Hash() *Hash {
	p, err := e.MarshalBinary()
	if err != nil {
		return NewHash()
	}
	return Sha(p)
}

func (e *ECBlockHeader) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 32 byte ECChainID
	buf.Write(e.ECChainID.Bytes())

	// 32 byte BodyHash
	buf.Write(e.BodyHash.Bytes())

	// 32 byte Previous Header Hash
	buf.Write(e.PrevHeaderHash.Bytes())

	// 32 byte Previous Full Hash
	buf.Write(e.PrevFullHash.Bytes())

	// 4 byte Directory Block Height
	if err := binary.Write(buf, binary.BigEndian, e.DBHeight); err != nil {
		return buf.Bytes(), err
	}

	// variable Header Expansion Size
	if _, err := WriteVarInt(buf,
		uint64(len(e.HeaderExpansionArea))); err != nil {
		return buf.Bytes(), err
	}

	// varable byte Header Expansion Area
	buf.Write(e.HeaderExpansionArea)

	// 8 byte Object Count
	if err := binary.Write(buf, binary.BigEndian, e.ObjectCount); err != nil {
		return buf.Bytes(), err
	}

	// 8 byte size of the Body
	if err := binary.Write(buf, binary.BigEndian, e.BodySize); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

func (e *ECBlockHeader) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	err := e.readUnmarshal(buf)
	if err != nil {
		return err
	}
	return nil
}

// readUnmarshal is a private method to read the correct lenth of bytes from a
// buffer and unmarshal the data
func (e *ECBlockHeader) readUnmarshal(buf *bytes.Buffer) error {
	hash := make([]byte, 32)

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.ECChainID.SetBytes(hash)
	}

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.BodyHash.SetBytes(hash)
	}

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.PrevHeaderHash.SetBytes(hash)
	}

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.PrevFullHash.SetBytes(hash)
	}

	if err := binary.Read(buf, binary.BigEndian, &e.DBHeight); err != nil {
		return err
	}

	// read the Header Expansion Area
	hesize := ReadVarInt(buf)
	e.HeaderExpansionArea = make([]byte, hesize)
	if _, err := buf.Read(e.HeaderExpansionArea); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &e.ObjectCount); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &e.BodySize); err != nil {
		return err
	}

	return nil
}
