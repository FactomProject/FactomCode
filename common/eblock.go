// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"io"
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

func MakeEBlock(echain *EChain, prev *EBlock) *EBlock {
	e := NewEBlock()
	e.Header.ChainID = echain.ChainID
	if prev != nil {
		e.Header.PrevKeyMR = prev.KeyMR()
		e.Header.PrevFullHash = prev.Hash()
	}
	e.Header.EBSequence = echain.NextBlockHeight
	return e
}

// NewEBlock returns a blank initialized Entry Block
func NewEBlock() *EBlock {
	e := new(EBlock)
	e.Header = NewEBlockHeader()
	e.Body = NewEBlockBody()
	return e
}

// AddEBEntry
func (e *EBlock) AddEBEntry(entry *Entry) error {
	e.Body.EBEntries = append(e.Body.EBEntries, entry.Hash())
	return nil
}

func (e *EBlock) AddEndOfMinuteMarker(m byte) {
	h := make([]byte, 32)
	h[len(h)-1] = m
	hash := NewHash()
	hash.SetBytes(h)
	e.Body.EBEntries = append(e.Body.EBEntries, hash)
}

// BuildHeader updates the Entry Block Header to include information about the
// Entry Block Body. BuildHeader should be run after the Entry Block Body has
// included all of its Entries.
func (e *EBlock) BuildHeader() error {
	e.Header.BodyMR = e.Body.MR()
	e.Header.EntryCount = uint32(len(e.Body.EBEntries))
	return nil
}

func (e *EBlock) Hash() *Hash {
	p, err := e.MarshalBinary()
	if err != nil {
		return NewHash()
	}
	return Sha(p)
}

func (e *EBlock) KeyMR() *Hash {
	// h(h(header) + BodyMR)
	e.BuildHeader()
	header, err := e.Header.MarshalBinary()
	if err != nil {
		return NewHash()
	}
	h1 := Sha(header)
	return Sha(append(h1.Bytes(), e.Header.BodyMR.Bytes()...))
}

func (e *EBlock) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	if err := e.BuildHeader(); err != nil {
		return buf.Bytes(), err
	}
	if p, err := e.Header.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	if p, err := e.Body.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func (e *EBlock) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	
	if err := e.Header.UnmarshalBinary(buf.Next(EBHeaderSize)); err != nil {
		return err
	}
	
	if err := e.Body.UnmarshalBinary(buf.Bytes()); err != nil {
		return err
	}
	
	return nil
}

type EBlockBody struct {
	EBEntries []*Hash
}

func NewEBlockBody() *EBlockBody {
	e := new(EBlockBody)
	e.EBEntries = make([]*Hash, 0)
	return e
}

func (e *EBlockBody) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	for _, v := range e.EBEntries {
		buf.Write(v.Bytes())
	}
	
	return buf.Bytes(), nil
}

func (e *EBlockBody) MR() *Hash {
	mrs := BuildMerkleTreeStore(e.EBEntries)
	r := mrs[len(mrs)-1]
	return r
}

func (e *EBlockBody) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)
	
	for {
		if _, err := buf.Read(hash); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		
		h := NewHash()
		h.SetBytes(hash)
		e.EBEntries = append(e.EBEntries, h)
	}
	
	return nil
}

type EBlockHeader struct {
	ChainID      *Hash
	BodyMR       *Hash
	PrevKeyMR    *Hash
	PrevFullHash *Hash
	EBSequence   uint32
	DBHeight     uint32
	EntryCount   uint32
}

func NewEBlockHeader() *EBlockHeader {
	e := new(EBlockHeader)
	e.ChainID = NewHash()
	e.BodyMR = NewHash()
	e.PrevKeyMR = NewHash()
	e.PrevFullHash = NewHash()
	return e
}

func (e *EBlockHeader) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// 32 byte ChainID
	buf.Write(e.ChainID.Bytes())

	// 32 byte Body MR
	buf.Write(e.BodyMR.Bytes())

	// 32 byte Previous Key MR
	buf.Write(e.PrevKeyMR.Bytes())

	// 32 byte Previous Full Hash
	buf.Write(e.PrevFullHash.Bytes())

	if err := binary.Write(buf, binary.BigEndian, e.EBSequence); err != nil {
		return buf.Bytes(), err
	}

	if err := binary.Write(buf, binary.BigEndian, e.DBHeight); err != nil {
		return buf.Bytes(), err
	}

	if err := binary.Write(buf, binary.BigEndian, e.EntryCount); err != nil {
		return buf.Bytes(), err
	}
	
	return buf.Bytes(), nil
}

func (e *EBlockHeader) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)
	
	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.ChainID.SetBytes(hash)
	}

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.BodyMR.SetBytes(hash)
	}

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.PrevKeyMR.SetBytes(hash)
	}

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.PrevFullHash.SetBytes(hash)
	}
	
	if err := binary.Read(buf, binary.BigEndian, &e.EBSequence); err != nil {
		return err
	}
	
	if err := binary.Read(buf, binary.BigEndian, &e.DBHeight); err != nil {
		return err
	}
	
	if err := binary.Read(buf, binary.BigEndian, &e.EntryCount); err != nil {
		return err
	}

	return nil
}
