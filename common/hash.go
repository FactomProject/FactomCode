// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/FactomProject/gocoding"
)

type Hash struct {
	Bytes []byte `json:"bytes"`
}

//Fixed size hash used for map, where byte slice wont work
type HashF [HASH_LENGTH]byte

func (h HashF) Hash() Hash {
	return Hash{Bytes: h[:]}
}

func (h *HashF) From(hash *Hash) {
	copy(h[:], hash.Bytes)
}

func NewHash() *Hash {
	h := new(Hash)
	h.Bytes = make([]byte, HASH_LENGTH)
	return h
}

//
// Creates a serial hash from a set of "entities"
//
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

// Function makes it easy to unmarshal Hashes.
// 
// x.HashThing, data, _ = UnmarshalHash(data)
//
func UnmarshalHash (data []byte) (newHash *Hash, newData []byte, err error) {
    newHash = make([]byte,HASH_LENGTH) 
    if(len(data)<HASH_LENGTH) {
        err = fmt.Errorf("Not enough data to unmarshal HASH")
        return
    }
    copy(newHash.Bytes,data)
    newData = data[HASH_LENGTH:]
}

func Sha(p []byte) (h *Hash) {
	sha := sha256.New()
	sha.Write(p)

	h = new(Hash)
	h.Bytes = sha.Sum(nil)
	return h
}

func (h *Hash) String() string {
	return hex.EncodeToString(h.Bytes)
}

func (h *Hash) ByteString() string {
	return string(h.Bytes)
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

/*****************************************************
 * Don't use Hash.MarshalBinary() or UnmarshalBinary in general.
 * Only use them where you want to call the functions to compute
 * Merkle Roots and Serial hashes
 *****************************************************/
func (h Hash)MarshalBinary() (data []byte, err error){
    return h.Bytes, nil
}

func (h Hash) UnmarshalBinary(data []byte) error {
    h.Bytes = make([]byte,HASH_LENGTH,HASH_LENGTH)
    copy(h.Bytes,data)
}