// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	//"fmt"
	"io"
	"github.com/FactomProject/FactomCode/common"
)


// InvVectHeight defines a factom inventory vector which is used to describe data,
// as specified by the Type field, that a peer wants, has, or does not have to
// another peer.
type InvVectHeight struct {
	Type InvType // Type of data
	Hash common.Hash // Hash of the data
	Height int64 // Height of the data; -1 means unknow height.
}

// NewInvVectHeight returns a new InvVectHeight using the provided type and hash.
func NewInvVectHeight(typ InvType, hash common.Hash, h int64) *InvVectHeight {
	return &InvVectHeight{
		Type: typ,
		Hash: hash,
		Height: h,
	}
}

// readInvVectHeight reads an encoded InvVectHeight from r depending on the protocol
// version.
func readInvVectHeight(r io.Reader, pver uint32, iv *InvVectHeight) error {
	err := readElements(r, &iv.Type, &iv.Hash, &iv.Height)
	if err != nil {
		return err
	}
	return nil
}

// writeInvVectHeight serializes an InvVectHeight to w depending on the protocol version.
func writeInvVectHeight(w io.Writer, pver uint32, iv *InvVectHeight) error {
	err := writeElements(w, iv.Type, &iv.Hash, &iv.Height)
	if err != nil {
		return err
	}
	return nil
}