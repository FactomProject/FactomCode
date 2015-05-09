// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"errors"
)

type Hash struct {
	Bytes []byte `json:"bytes"`
}

func NewHash() *Hash {
	h := new(Hash)
	h.Bytes = make([]byte, HASH_LENGTH)
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
	h.Bytes = sha.Sum(nil)
	return
}

func (h *Hash) MarshalBinary() ([]byte, error) {
	if h.Bytes == nil || len(h.Bytes) != HASH_LENGTH {
		return nil, errors.New("(h *Hash) MarshalBinary() Invalid Hash byte stream: " + string(h.Bytes))
	}	
	var buf bytes.Buffer
	buf.Write(h.Bytes)
	return buf.Bytes(), nil
}

func (h *Hash) UnmarshalBinary(p []byte) error {
	h.Bytes = make([]byte, HASH_LENGTH)	
	copy(h.Bytes, p)
	return nil
}

func (h *Hash) GetBytes() []byte {
	newHash := make([]byte, HASH_LENGTH)
	copy(newHash, h.Bytes)

	return newHash
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not HASH_LENGTH.
func (hash *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HASH_LENGTH {
		return fmt.Errorf("invalid sha length of %v, want %v", nhlen, HASH_LENGTH)
	}

	hash.Bytes = make([]byte, HASH_LENGTH)
	copy(hash.Bytes, newHash)
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
	h.Bytes = sha.Sum(nil)
	return h
}

// Convert a hash into a string with hex encoding
func (h *Hash) String() string {
	if h == nil {
		return hex.EncodeToString(nil)
	} else {
		return hex.EncodeToString(h.Bytes)
	}
}

func (h *Hash) ByteString() string {
	return string(h.Bytes)
}

func HexToHash(hexStr string) (h *Hash, err error) {
	h = new(Hash)
	h.Bytes, err = hex.DecodeString(hexStr)
	return h, err
}

// String returns the ShaHash in the standard bitcoin big-endian form.
func (h *Hash) BTCString() string {
	hashstr := ""
	hash := h.Bytes
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

	if bytes.Compare(a.Bytes, b.Bytes) == 0 {
		return true
	}

	return false
}
