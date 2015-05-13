// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
)

const (
	ECIDServerIndexNumber byte = iota
	ECIDMinuteNumber
	ECIDChainCommit
	ECIDEntryCommit
	ECIDBalanceIncrease
)

// The Entry Credit Block consists of a header and a body. The body is composed
// of primarily Commits and balance increases with minute markers and server
// markers distributed throughout the body.
type ECBlock struct {
	Header *ECBlockHeader
	Body   []ECBlockEntry
}

func (e *ECBlock) AddEntry(n ECBlockEntry) {
	e.Body = append(e.Body, n)
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
		buf.WriteByte(v.Type())
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

type ECBlockEntry interface {
	Type() byte
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
}

func (e *ECBlockHeader) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(e.ECChainID.Bytes)
	buf.Write(e.BodyHash.Bytes)
	buf.Write(e.PrevKeyMR.Bytes)
	buf.Write(e.PrevHash3.Bytes)
	if err := binary.Write(buf, binary.BigEndian, e.DBHeight); err != nil {
		return buf.Bytes(), err
	}
	buf.Write(e.SegmentsMR.Bytes)
	buf.Write(e.BalanceCommit.Bytes)
	if err := binary.Write(buf, binary.BigEndian, e.ObjectCount); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}
