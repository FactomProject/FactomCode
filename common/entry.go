// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// An Entry is the element which carries user data
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#entry
type Entry struct {
	Version uint8
	ChainID *Hash
	ExtIDs  [][]byte
	Content []byte
}

func NewEntry() *Entry {
	e := new(Entry)
	e.ChainID = NewHash()
	e.ExtIDs = make([][]byte, 0)
	e.Content = make([]byte, 0)
	return e
}

// NewChainID generates a ChainID from an entry. ChainID = Sha(Sha(ExtIDs[0]) +
// Sha(ExtIDs[1] + ... + Sha(ExtIDs[n]))
func NewChainID(e *Entry) *Hash {
	id := new(Hash)
	sum := sha256.New()
	for _, v := range e.ExtIDs {
		x := sha256.Sum256(v)
		sum.Write(x[:])
	}
	copy(id.Bytes(), sum.Sum(nil))

	return id
}

func (e *Entry) Hash() *Hash {
	h := NewHash()
	a, err := e.MarshalBinary()
	if err != nil {
		return h
	}
	b := sha3.Sum256(a)
	c := append(a, b[:]...)
	d := sha256.Sum256(c)
	copy(h.Bytes(), d[:])

	return h
}

func (e *Entry) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1 byte Version
	if err := binary.Write(buf, binary.BigEndian, e.Version); err != nil {
		return buf.Bytes(), err
	}

	// 32 byte ChainID
	buf.Write(e.ChainID.Bytes())

	// ExtIDs
	if ext, err := e.MarshalExtIDsBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		// 2 byte size of ExtIDs
		if err := binary.Write(buf, binary.BigEndian, int16(len(ext)));
			err != nil {
			return buf.Bytes(), err
		}

		// binary ExtIDs
		buf.Write(ext)
	}

	// Content
	buf.Write(e.Content)

	return buf.Bytes(), nil
}

// MarshalExtIDsBinary marshals the ExtIDs into a []byte containing a series of
// 2 byte size of each ExtID followed by the ExtID.
func (e *Entry) MarshalExtIDsBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, x := range e.ExtIDs {
		// 2 byte size of the ExtID
		if err := binary.Write(buf, binary.BigEndian, uint16(len(x)));
			err != nil {
			return buf.Bytes(), err
		}

		// ExtID bytes
		buf.Write(x)
	}

	return buf.Bytes(), nil
}

func (e *Entry) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewBuffer(data)

	// 1 byte Version
	if x, err := buf.ReadByte(); err != nil {
		return err
	} else {
		e.Version = x
	}

	// 32 byte ChainID
	e.ChainID = NewHash()
	if _, err := buf.Read(e.ChainID.Bytes()); err != nil {
		return err
	}

	// 2 byte size of ExtIDs
	var extSize uint16
	if err := binary.Read(buf, binary.BigEndian, &extSize); err != nil {
		return err
	}

	// ExtIDs
	for i := extSize; i > 0; {
		var xsize int16
		binary.Read(buf, binary.BigEndian, &xsize)
		i -= 2

		x := make([]byte, xsize)
		if n, err := buf.Read(x); err != nil {
			return err
		} else {
			if c := cap(x); n != c {
				return fmt.Errorf("Could not read ExtID: Read %d bytes of %d\n",
					n, c)
			}
			e.ExtIDs = append(e.ExtIDs, x)
			i -= uint16(n)
		}
	}

	// Content
	e.Content = buf.Bytes()

	return nil
}
