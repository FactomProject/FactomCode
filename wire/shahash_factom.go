// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"github.com/FactomProject/FactomCode/common"
)

// Convert wire.ShaHash into factom.common.Hash
func (hash *ShaHash) ToFactomHash() *common.Hash {
	commonhash := new(common.Hash)
	commonhash.SetBytes(hash.Bytes())

	return commonhash
}

// Convert factom.common.hash into a wire.ShaHash
func FactomHashToShaHash(ftmHash *common.Hash) *ShaHash {

	h, _ := NewShaHash(ftmHash.Bytes())
	return h

}
