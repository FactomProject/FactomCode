// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// based on btcsuite's shahash.go

// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// note: sha3.Sum256() is the main routine to use, similarlyto DoubleSha256() or common/hash.go sha256.New()

package wire

import (
	"bytes"
	"encoding/hex"
	//	"encoding/json"
	"fmt"
	//	"github.com/FactomProject/golangcrypto/sha3"
)

// Size of array used to store sha hashes.  See Sha3Hash.
const Hash3Size = 32

// MaxHashStringSize is the maximum length of a Sha3Hash hash string.
const Max3HashStringSize = Hash3Size * 2

// ErrHashStrSize describes an error that indicates the caller specified a hash
// string that has too many characters.
var Err3HashStrSize = fmt.Errorf("max hash string length for sha3 is %v bytes", Max3HashStringSize)

// Sha3Hash is used in several of the bitcoin messages and common structures.  It
// typically represents the double sha256 of data.
type Sha3Hash [Hash3Size]byte

// String returns the Sha3Hash as the hexadecimal string of the byte-reversed
// hash.
func (hash Sha3Hash) String() string {
	for i := 0; i < Hash3Size/2; i++ {
		hash[i], hash[Hash3Size-1-i] = hash[Hash3Size-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

// Bytes returns the bytes which represent the hash as a byte slice.
func (hash *Sha3Hash) Bytes() []byte {
	newHash := make([]byte, Hash3Size)
	copy(newHash, hash[:])

	return newHash
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not Hash3Size.
func (hash *Sha3Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != Hash3Size {
		return fmt.Errorf("invalid sha3 length of %v, want %v", nhlen,
			Hash3Size)
	}
	copy(hash[:], newHash[0:Hash3Size])

	return nil
}

// IsEqual returns true if target is the same as hash.
func (hash *Sha3Hash) IsEqual(target *Sha3Hash) bool {
	return bytes.Equal(hash[:], target[:])
}

// NewSha3Hash returns a new Sha3Hash from a byte slice.  An error is returned if
// the number of bytes passed in is not Hash3Size.
func NewSha3Hash(newHash []byte) (*Sha3Hash, error) {
	var sh Sha3Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// NewSha3HashFromStr creates a Sha3Hash from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Sha3Hash.
func NewSha3HashFromStr(hash string) (*Sha3Hash, error) {
	// Return error if hash string is too long.
	if len(hash) > Max3HashStringSize {
		return nil, Err3HashStrSize
	}

	// Hex decoder expects the hash to be a multiple of two.
	if len(hash)%2 != 0 {
		hash = "0" + hash
	}

	// Convert string hash to bytes.
	buf, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}

	// Un-reverse the decoded bytes, copying into in leading bytes of a
	// Sha3Hash.  There is no need to explicitly pad the result as any
	// missing (when len(buf) < Hash3Size) bytes from the decoded hex string
	// will remain zeros at the end of the Sha3Hash.
	var ret Sha3Hash
	blen := len(buf)
	mid := blen / 2
	if blen%2 != 0 {
		mid++
	}
	blen--
	for i, b := range buf[:mid] {
		ret[i], ret[blen-i] = buf[blen-i], b
	}
	return &ret, nil
}
