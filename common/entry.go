// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
)

// An Entry is the element which carries user data
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#entry
type Entry struct {
	Printable          `json:"-"`
	BinaryMarshallable `json:"-"`

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
	id.SetBytes(sum.Sum(nil))

	return id
}

func (e *Entry) Hash() *Hash {
	h := NewHash()
	entry, err := e.MarshalBinary()
	if err != nil {
		return h
	}

	h1 := sha512.Sum512(entry)
	h2 := sha256.Sum256(append(h1[:], entry[:]...))
	h.SetBytes(h2[:])
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
		if err := binary.Write(buf, binary.BigEndian, int16(len(ext))); err != nil {
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
		if err := binary.Write(buf, binary.BigEndian, uint16(len(x))); err != nil {
			return buf.Bytes(), err
		}

		// ExtID bytes
		buf.Write(x)
	}

	return buf.Bytes(), nil
}

func (e *Entry) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	// 1 byte Version
	var b byte
	if b, err = buf.ReadByte(); err != nil {
		return
	} else {
		e.Version = b
	}

	// 32 byte ChainID
	e.ChainID = NewHash()
	if _, err = buf.Read(hash); err != nil {
		return
	} else if err = e.ChainID.SetBytes(hash); err != nil {
		return
	}

	// 2 byte size of ExtIDs
	var extSize uint16
	if err = binary.Read(buf, binary.BigEndian, &extSize); err != nil {
		return
	}

	// ExtIDs
	for i := int16(extSize); i > 0; {
		var xsize int16
		binary.Read(buf, binary.BigEndian, &xsize)
		i -= 2
		if i < 0 {
			err = fmt.Errorf("Error parsing external IDs")
			return
		}
		x := make([]byte, xsize)
		var n int
		if n, err = buf.Read(x); err != nil {
			return
		} else {
			if c := cap(x); n != c {
				err = fmt.Errorf("Could not read ExtID: Read %d bytes of %d\n", n, c)
				return
			}
			e.ExtIDs = append(e.ExtIDs, x)
			i -= int16(n)
			if i < 0 {
				err = fmt.Errorf("Error parsing external IDs")
				return
			}
		}
	}

	// Content
	e.Content = buf.Bytes()

	return
}

func (e *Entry) UnmarshalBinary(data []byte) (err error) {
	_, err = e.UnmarshalBinaryData(data)
	return
}

func (e *Entry) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *Entry) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *Entry) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *Entry) Spew() string {
	return Spew(e)
}
