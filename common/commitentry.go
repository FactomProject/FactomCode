// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	ed "github.com/FactomProject/ed25519"
)

const (
	// CommitEntrySize = 1 + 6 + 32 + 1 + 32 + 64
	CommitEntrySize int = 136
)

type CommitEntry struct {
	Version   uint8
	MilliTime *[6]byte
	EntryHash *Hash
	Credits   uint8
	ECPubKey  *[32]byte
	Sig       *[64]byte
}

var _ Printable = (*CommitEntry)(nil)
var _ BinaryMarshallable = (*CommitEntry)(nil)
var _ ShortInterpretable = (*CommitEntry)(nil)
var _ ECBlockEntry = (*CommitEntry)(nil)

func (c *CommitEntry) MarshalledSize() uint64 {
	return uint64(CommitEntrySize)
}

func NewCommitEntry() *CommitEntry {
	c := new(CommitEntry)
	c.Version = 0
	c.MilliTime = new([6]byte)
	c.EntryHash = NewHash()
	c.Credits = 0
	c.ECPubKey = new([32]byte)
	c.Sig = new([64]byte)
	return c
}

func (e *CommitEntry) Hash() *Hash {
	bin, err := e.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return Sha(bin)
}

func (b *CommitEntry) IsInterpretable() bool {
	return false
}

func (b *CommitEntry) Interpret() string {
	return ""
}

// CommitMsg returns the binary marshaled message section of the CommitEntry
// that is covered by the CommitEntry.Sig.
func (c *CommitEntry) CommitMsg() []byte {
	p, err := c.MarshalBinary()
	if err != nil {
		return []byte{byte(0)}
	}
	return p[:len(p)-64-32]
}

// Return the timestamp in milliseconds.
func (c *CommitEntry) GetMilliTime() int64 {
	a := make([]byte, 2, 8)
	a = append(a, c.MilliTime[:]...)
	milli := int64(binary.BigEndian.Uint64(a))
	return milli
}

// InTime checks the CommitEntry.MilliTime and returns true if the timestamp is
// whitin +/- 24 hours of the current time.
func (c *CommitEntry) InTime() bool {
	now := time.Now()
	sec := c.GetMilliTime() / 1000
	t := time.Unix(sec, 0)

	return t.After(now.Add(-24*time.Hour)) && t.Before(now.Add(24*time.Hour))
}

func (c *CommitEntry) IsValid() bool {

	//double check the credits in the commit
	if c.Credits < 1 || c.Version != 0 {
		return false
	}	
	
	return ed.VerifyCanonical(c.ECPubKey, c.CommitMsg(), c.Sig)
}

func (c *CommitEntry)GetHash() *Hash {
	h,_ :=	c.MarshalBinary()
	return Sha(h)
}


func (c *CommitEntry) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1 byte Version
	if err := binary.Write(buf, binary.BigEndian, c.Version); err != nil {
		return buf.Bytes(), err
	}

	// 6 byte MilliTime
	buf.Write(c.MilliTime[:])

	// 32 byte Entry Hash
	buf.Write(c.EntryHash.Bytes())

	// 1 byte number of Entry Credits
	if err := binary.Write(buf, binary.BigEndian, c.Credits); err != nil {
		return buf.Bytes(), err
	}

	// 32 byte Public Key
	buf.Write(c.ECPubKey[:])

	// 64 byte Signature
	buf.Write(c.Sig[:])

	return buf.Bytes(), nil
}

func (c *CommitEntry) ECID() byte {
	return ECIDEntryCommit
}

func (c *CommitEntry) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	var b byte
	var p []byte
	// 1 byte Version
	if b, err = buf.ReadByte(); err != nil {
		return
	} else {
		c.Version = uint8(b)
	}

	if buf.Len() < 6 {
		err = io.EOF
		return
	}

	// 6 byte MilliTime
	if p = buf.Next(6); p == nil {
		err = fmt.Errorf("Could not read MilliTime")
		return
	} else {
		copy(c.MilliTime[:], p)
	}

	// 32 byte Entry Hash
	if _, err = buf.Read(hash); err != nil {
		return
	} else if err = c.EntryHash.SetBytes(hash); err != nil {
		return
	}

	// 1 byte number of Entry Credits
	if b, err = buf.ReadByte(); err != nil {
		return
	} else {
		c.Credits = uint8(b)
	}

	if buf.Len() < 32 {
		err = io.EOF
		return
	}

	// 32 byte Public Key
	if p = buf.Next(32); p == nil {
		err = fmt.Errorf("Could not read ECPubKey")
		return
	} else {
		copy(c.ECPubKey[:], p)
	}

	if buf.Len() < 64 {
		err = io.EOF
		return
	}

	// 64 byte Signature
	if p = buf.Next(64); p == nil {
		err = fmt.Errorf("Could not read Sig")
		return
	} else {
		copy(c.Sig[:], p)
	}

	newData = buf.Bytes()

	return
}

func (c *CommitEntry) UnmarshalBinary(data []byte) (err error) {
	_, err = c.UnmarshalBinaryData(data)
	return
}

func (e *CommitEntry) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *CommitEntry) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *CommitEntry) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *CommitEntry) Spew() string {
	return Spew(e)
}
