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
	// CommitChainSize = 1+6+32+32+32+1+32+64
	CommitChainSize int = 200
)

type CommitChain struct {
	Version     uint8
	MilliTime   *[6]byte
	ChainIDHash *Hash
	Weld        *Hash
	EntryHash   *Hash
	Credits     uint8
	ECPubKey    *[32]byte
	Sig         *[64]byte
}

var _ Printable = (*CommitChain)(nil)
var _ BinaryMarshallable = (*CommitChain)(nil)

func (c *CommitChain) MarshalledSize() uint64 {
	return uint64(CommitChainSize)
}

func NewCommitChain() *CommitChain {
	c := new(CommitChain)
	c.Version = 0
	c.MilliTime = new([6]byte)
	c.ChainIDHash = NewHash()
	c.Weld = NewHash()
	c.EntryHash = NewHash()
	c.Credits = 0
	c.ECPubKey = new([32]byte)
	c.Sig = new([64]byte)
	return c
}

// CommitMsg returns the binary marshaled message section of the CommitEntry
// that is covered by the CommitEntry.Sig.
func (c *CommitChain) CommitMsg() []byte {
	p, err := c.MarshalBinary()
	if err != nil {
		return []byte{byte(0)}
	}
	return p[:len(p)-64-32]
}

// Return the timestamp in milliseconds.
func (c *CommitChain) GetMilliTime() int64 {
	a := make([]byte, 2, 8)
	a = append(a, c.MilliTime[:]...)
	milli := int64(binary.BigEndian.Uint64(a))
	return milli
}

// InTime checks the CommitEntry.MilliTime and returns true if the timestamp is
// whitin +/- 24 hours of the current time.
// TODO
func (c *CommitChain) InTime() bool {
	now := time.Now()
	sec := c.GetMilliTime() / 1000
	t := time.Unix(sec, 0)

	return t.After(now.Add(-24*time.Hour)) && t.Before(now.Add(24*time.Hour))
}

func (c *CommitChain) IsValid() bool {
	
	//double check the credits in the commit
	if c.Credits < 1 || c.Version != 0 {
		return false
	}
	
	return ed.VerifyCanonical(c.ECPubKey, c.CommitMsg(), c.Sig)
}

func (c *CommitChain) GetHash() *Hash {
	data, _ := c.MarshalBinary()
	return Sha(data)
}

func (c *CommitChain) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1 byte Version
	if err := binary.Write(buf, binary.BigEndian, c.Version); err != nil {
		return buf.Bytes(), err
	}

	// 6 byte MilliTime
	buf.Write(c.MilliTime[:])

	// 32 byte double sha256 hash of the ChainID
	buf.Write(c.ChainIDHash.Bytes())

	// 32 byte Commit Weld sha256(sha256(Entry Hash + ChainID))
	buf.Write(c.Weld.Bytes())

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

func (c *CommitChain) ECID() byte {
	return ECIDChainCommit
}

func (c *CommitChain) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	// 1 byte Version
	var b byte
	var p []byte
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

	// 32 byte ChainIDHash
	if _, err = buf.Read(hash); err != nil {
		return
	} else if err = c.ChainIDHash.SetBytes(hash); err != nil {
		return
	}

	// 32 byte Weld
	if _, err = buf.Read(hash); err != nil {
		return
	} else if err = c.Weld.SetBytes(hash); err != nil {
		return
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
	if p := buf.Next(32); p == nil {
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
	if p := buf.Next(64); p == nil {
		err = fmt.Errorf("Could not read Sig")
		return
	} else {
		copy(c.Sig[:], p)
	}

	newData = buf.Bytes()

	return
}

func (c *CommitChain) UnmarshalBinary(data []byte) (err error) {
	_, err = c.UnmarshalBinaryData(data)
	return
}

func (e *CommitChain) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *CommitChain) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *CommitChain) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *CommitChain) Spew() string {
	return Spew(e)
}
