// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"fmt"

	ed "github.com/agl/ed25519"
)

type CommitEntry struct {
	Version   uint8
	MilliTime *[6]byte
	EntryHash *Hash
	Credits   uint8
	ECPubKey  *[32]byte
	Sig       *[64]byte
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

func (c *CommitEntry) IsValid() bool {
	return ed.Verify(c.ECPubKey, c.CommitMsg(), c.Sig)
}

func (c *CommitEntry) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1 byte Version
	if err := binary.Write(buf, binary.BigEndian, c.Version); err != nil {
		return buf.Bytes(), err
	}

	// 6 byte MilliTime
	if _, err := buf.Write(c.MilliTime[:]); err != nil {
		return buf.Bytes(), err
	}

	// 32 byte Entry Hash
	if _, err := buf.Write(c.EntryHash.Bytes); err != nil {
		return buf.Bytes(), err
	}

	// 1 byte number of Entry Credits
	if err := binary.Write(buf, binary.BigEndian, c.Credits); err != nil {
		return buf.Bytes(), err
	}

	// 32 byte Public Key
	if _, err := buf.Write(c.ECPubKey[:]); err != nil {
		return buf.Bytes(), err
	}

	// 64 byte Signature
	if _, err := buf.Write(c.Sig[:]); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

func (c *CommitEntry) UnmarshalBinary(d []byte) (err error) {
	buf := bytes.NewBuffer(d)

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

	// 32 byte Entry Hash
	if p := buf.Next(32); p == nil {
		return fmt.Errorf("Could not read EntryHash")
	} else {
		copy(c.EntryHash.Bytes, p)
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
