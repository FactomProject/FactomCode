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

	// ecBlockHeaderSize 32+32+32+32+4+32+32+8
	ecBlockHeaderSize = 204
)

// The Entry Credit Block consists of a header and a body. The body is composed
// of primarily Commits and balance increases with minute markers and server
// markers distributed throughout.
type ECBlock struct {
	Header *ECBlockHeader
	Body   []ECBlockEntry
}

func NewECBlock() *ECBlock {
	e := new(ECBlock)
	e.Header = NewECBlockHeader()
	e.Body = make([]ECBlockEntry, 0)
	return e
}

func (e *ECBlock) AddEntries(entries ...ECBlockEntry) {
	e.Body = append(e.Body, entries...)
	e.Header.ObjectCount = uint64(len(e.Body))
}

func (e *ECBlock) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Header
	if p, err := e.Header.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	// Body of ECBlockEntries
	for _, v := range e.Body {
		p, err := v.MarshalBinary()
		if err != nil {
			return buf.Bytes(), err
		}
		buf.WriteByte(v.ECID())
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

/* TODO
func (e *ECBlock) GenerateHeader() {
	// Marshal the Body
	buf := new(bytes.Buffer)
	
	for _, v := range e.Body {
		p, err := v.MarshalBinary()
		if err != nil {
			// something
		}
		buf.WriteByte(v.ECID())
		buf.Write(p)
	}
	
	body := buf.Bytes()
	
	e.Header.BodyHash.Bytes = sha256.Sum256(body)[:]
	e.Header.ObjectCount = len(e.Body)
	e.Header.BodySize = len(body)
}
*/

func (e *ECBlock) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	
	// Unmarshal Header
	if p := buf.Next(ecBlockHeaderSize); len(p) != ecBlockHeaderSize {
		return fmt.Errorf("Entry Block is smaller than ecBlockHeaderSize")
	} else {
		if err := e.Header.UnmarshalBinary(p); err != nil {
			return err
		}
	}
	
	// Unmarshal Body
	for {
		if id, err := buf.ReadByte(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else {
			switch id {
			case ECIDServerIndexNumber:
			case ECIDMinuteNumber:
			case ECIDChainCommit:
				x := NewCommitChain()
				err := x.UnmarshalBinary(buf.Next(CommitChainSize))
				if err != nil {
					return err
				}
				e.AddEntries(x)
			case ECIDEntryCommit:
				x := NewCommitEntry()
				err := x.UnmarshalBinary(buf.Next(CommitEntrySize))
				if err != nil {
					return err
				}
				e.AddEntries(x)
			case ECIDBalanceIncrease:
			default:
				return fmt.Errorf("Unsupported ECID: %x\n", id)
			}
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
	ECChainID     *Hash
	BodyHash      *Hash
	PrevKeyMR     *Hash
	PrevHash3     *Hash
	DBHeight      uint32
	SegmentsMR    *Hash
	BalanceCommit *Hash
	ObjectCount   uint64
	BodySize      uint64
}

func NewECBlockHeader() *ECBlockHeader {
	h := new(ECBlockHeader)
	h.ECChainID = NewHash()
	h.BodyHash = NewHash()
	h.PrevKeyMR = NewHash()
	h.PrevHash3 = NewHash()
	h.DBHeight = 0
	h.SegmentsMR = NewHash()
	h.BalanceCommit = NewHash()
	h.ObjectCount = 0
	return h
}

func (e *ECBlockHeader) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 32 byte ECChainID
	buf.Write(e.ECChainID.Bytes)
	
	// 32 byte BodyHash
	buf.Write(e.BodyHash.Bytes)
	
	// 32 byte Previous KeyMR
	buf.Write(e.PrevKeyMR.Bytes)
	
	// 32 byte Previous Hash
	buf.Write(e.PrevHash3.Bytes)
	
	// 4 byte Directory Block Height
	if err := binary.Write(buf, binary.BigEndian, e.DBHeight); err != nil {
		return buf.Bytes(), err
	}
	
	// 32 byte SegmentsMR
	buf.Write(e.SegmentsMR.Bytes)
	
	// 32 byte Balance Commit
	buf.Write(e.BalanceCommit.Bytes)
	
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
	
	if _, err := buf.Read(e.ECChainID.Bytes); err != nil {
		return err
	}
	if _, err := buf.Read(e.BodyHash.Bytes); err != nil {
		return err
	}
	if _, err := buf.Read(e.PrevKeyMR.Bytes); err != nil {
		return err
	}
	if _, err := buf.Read(e.PrevHash3.Bytes); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &e.DBHeight); err != nil {
		return err
	}
	if _, err := buf.Read(e.SegmentsMR.Bytes); err != nil {
		return err
	}
	if _, err := buf.Read(e.BalanceCommit.Bytes); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &e.ObjectCount); err != nil {
		return err
	}
	
	return nil
}
