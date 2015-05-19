// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	
	"golang.org/x/crypto/sha3"
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

func NextECBlock(p *ECBlock) *ECBlock {
	e := NewECBlock()
	e.Header.PrevKeyMR = p.KeyMR()
	e.Header.PrevHash3 = p.Hash3()
	e.Header.DBHeight = p.Header.DBHeight + 1
	return e
}

func (e *ECBlock) AddEntry(entries ...ECBlockEntry) {
	e.Body.Entries = append(e.Body.Entries, entries...)
}

// Hash3 returns the sha3-256 checksum of the previous Entry Credit Block from
// the Header to the end of the Body
func (e *ECBlock) Hash3() *Hash {
	r := NewHash()
	
	p, err := e.MarshalBinary()
	if err != nil {
		return r
	}
	
	sum := sha3.Sum256(p)
	copy(r.Bytes, sum[:])
	return r
}

// KeyMR returns a hash of the serialized Block Header + the serialized Body.
func (e *ECBlock) KeyMR() *Hash {
	r := NewHash()
	p := make([]byte, 0)
	
	head, err := e.Header.MarshalBinary()
	if err != nil {
		return r
	}
	p = append(p, head...)
	body, err := e.Body.MarshalBinary()
	if err != nil {
		return r
	}
	p = append(p, body...)
	
	return Sha(p)
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
	if err := e.Body.UnmarshalBinary(buf.Bytes()); err != nil {
		return err
	}
	
	return nil
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

type ECChain struct {
	ChainID         *Hash
	Name            [][]byte
	NextBlock       *ECBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

func NewECChain() *ECChain {
	c := new(ECChain)
	c.ChainID = NewHash()
	copy(c.ChainID.Bytes, EC_CHAINID)
	c.Name = make([][]byte, 0)
	return c
}

func (c *ECChain) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	if p, err := c.ChainID.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	binary.Write(buf, binary.BigEndian, uint64(len(c.Name)))

	for _, v := range c.Name {
		binary.Write(buf, binary.BigEndian, uint64(len(v)))
		buf.Write(v)
	}

	return buf.Bytes(), nil
}

func (c *ECChain) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	
	if _, err := buf.Read(c.ChainID.Bytes); err != nil {
		return err
	}

	count := uint64(0)
	if err := binary.Read(buf, binary.BigEndian, count); err != nil {
		return err
	}
	c.Name = make([][]byte, count)

	for _, name := range c.Name {
		var l uint64
		if err := binary.Read(buf, binary.BigEndian, &l); err != nil {
			return err
		}
		name = make([]byte, l)
		if _, err := buf.Read(name); err != nil {
			return err
		}
	}

	return nil
}
