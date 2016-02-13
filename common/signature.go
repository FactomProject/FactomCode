// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"encoding/hex"

	"github.com/FactomProject/ed25519"
)

type DetachedSignature [ed25519.SignatureSize]byte
type DetachedPublicKey [ed25519.PublicKeySize]byte

//Signature has signed data and its corresponsing PublicKey
type Signature struct {
	Pub PublicKey
	Sig *[ed25519.SignatureSize]byte
}

func (sig Signature) Key() []byte {
	return (*sig.Pub.Key)[:]
}

func (sig *Signature) DetachSig() *DetachedSignature {
	return (*DetachedSignature)(sig.Sig)
}

func (ds *DetachedSignature) String() string {
	return hex.EncodeToString(ds[:])
}

func UnmarshalBinarySignature(data []byte) (sig Signature) {
	sig.Pub.Key = new([32]byte)
	sig.Sig = new([64]byte)
	copy(sig.Pub.Key[:], data[:32])
	data = data[32:]
	copy(sig.Sig[:], data[:64])
	return
}

func MarshalBinarySignature(sig Signature) (data [96]byte) {
	copy(data[:32], sig.Pub.Key[:])
	copy(data[32:], sig.Sig[:])
	return
}
