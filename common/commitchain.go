// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"fmt"

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

// InTime checks the CommitEntry.MilliTime and returns true if the timestamp is
// whitin +/- 24 hours of the current time.
// TODO
func (c *CommitChain) InTime() bool {
	return true
}

func (c *CommitChain) IsValid() bool {
	return ed.Verify(c.ECPubKey, c.CommitMsg(), c.Sig)
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

func (c *CommitChain) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	// 1 byte Version
	if p, err := buf.ReadByte(); err != nil {
		return err
	} else {
		c.Version = uint8(p)
	}

	// 6 byte MilliTime
	if p := buf.Next(6); p == nil {
		return fmt.Errorf("Could not read MilliTime")
	} else {
		copy(c.MilliTime[:], p)
	}

	// 32 byte ChainIDHash
	if _, err := buf.Read(hash); err != nil {
		return err
	} else if err := c.ChainIDHash.SetBytes(hash); err != nil {
		return err
	}

	// 32 byte Weld
	if _, err := buf.Read(hash); err != nil {
		return err
	} else if err := c.Weld.SetBytes(hash); err != nil {
		return err
	}

	// 32 byte Entry Hash
	if _, err := buf.Read(hash); err != nil {
		return err
	} else if err := c.EntryHash.SetBytes(hash); err != nil {
		return err
	}

	// 1 byte number of Entry Credits
	if p, err := buf.ReadByte(); err != nil {
		return err
	} else {
		c.Credits = uint8(p)
	}

	// 32 byte Public Key
	if p := buf.Next(32); p == nil {
		return fmt.Errorf("Could not read ECPubKey")
	} else {
		copy(c.ECPubKey[:], p)
	}

	// 64 byte Signature
	if p := buf.Next(64); p == nil {
		return fmt.Errorf("Could not read Sig")
	} else {
		copy(c.Sig[:], p)
	}

	return nil
}
