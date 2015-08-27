// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

type Hash struct {
	bytes [HASH_LENGTH]byte
}

var _ Printable = (*Hash)(nil)
var _ BinaryMarshallable = (*Hash)(nil)

func (c *Hash) MarshalledSize() uint64 {
	return uint64(HASH_LENGTH)
}

func (h *Hash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h.bytes[:])), nil
}

func (h *Hash) UnmarshalText(b []byte) error {
	p, err := hex.DecodeString(string(b))
	if err != nil {
		return err
	}
	copy(h.bytes[:], p)
	return nil
}

func (h *Hash) Bytes() []byte {
	return h.GetBytes()
}

func NewHash() *Hash {
	h := new(Hash)
	return h
}

func CreateHash(entities ...BinaryMarshallable) (h *Hash, err error) {
	sha := sha256.New()
	h = new(Hash)
	for _, entity := range entities {
		data, err := entity.MarshalBinary()
		if err != nil {
			return nil, err
		}
		sha.Write(data)
	}
	copy(h.bytes[:], sha.Sum(nil))
	return
}

func (h *Hash) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(h.bytes[:])
	return buf.Bytes(), nil
}

func (h *Hash) UnmarshalBinaryData(p []byte) (newData []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()
	copy(h.bytes[:], p)
	newData = p[HASH_LENGTH:]
	return
}

func (h *Hash) UnmarshalBinary(p []byte) (err error) {
	_, err = h.UnmarshalBinaryData(p)
	return
}

func (h *Hash) GetBytes() []byte {
	newHash := make([]byte, HASH_LENGTH)
	copy(newHash, h.bytes[:])

	return newHash
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not HASH_LENGTH.
func (hash *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HASH_LENGTH {
		return fmt.Errorf("invalid sha length of %v, want %v", nhlen, HASH_LENGTH)
	}
	copy(hash.bytes[:], newHash)
	return nil
}

// NewShaHash returns a new ShaHash from a byte slice.  An error is returned if
// the number of bytes passed in is not HASH_LENGTH.
func NewShaHash(newHash []byte) (*Hash, error) {
	var sh Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// Create a Sha256 Hash from a byte array
func Sha(p []byte) (h *Hash) {
	sha := sha256.New()
	sha.Write(p)

	h = new(Hash)
	copy(h.bytes[:], sha.Sum(nil))
	return h
}

// Create a Sha512[:256] Hash from a byte array
func Sha512Half(p []byte) (h *Hash) {
	sha := sha512.New()
	sha.Write(p)

	h = new(Hash)
	copy(h.bytes[:], sha.Sum(nil)[:32])
	return h
}

// Convert a hash into a string with hex encoding
func (h *Hash) String() string {
	if h == nil {
		return hex.EncodeToString(nil)
	} else {
		return hex.EncodeToString(h.bytes[:])
	}
}

func (h *Hash) ByteString() string {
	return string(h.bytes[:])
}

func HexToHash(hexStr string) (h *Hash, err error) {
	h = new(Hash)
	v, err := hex.DecodeString(hexStr)
	err = h.SetBytes(v)
	return h, err
}

// String returns the ShaHash in the standard bitcoin big-endian form.
func (h *Hash) BTCString() string {
	hashstr := ""
	hash := h.bytes
	for i := range hash {
		hashstr += fmt.Sprintf("%02x", hash[HASH_LENGTH-1-i])
	}

	return hashstr
}

// Compare two Hashes
func (a *Hash) IsSameAs(b *Hash) bool {
	if a == nil || b == nil {
		return false
	}

	if a.bytes == b.bytes {
		return true
	}

	return false
}

// Is the hash a minute marker (the last byte indicates the minute number)
func (h *Hash) IsMinuteMarker() bool {
	

	if bytes.Equal(h.bytes[:31], ZERO_HASH[:31]) {
		return true
	}

	return false
}

func (e *Hash) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *Hash) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *Hash) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *Hash) Spew() string {
	return Spew(e)
}

// shad Double Sha256 Hash; sha256(sha256(data))
func DoubleSha(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:]
}
