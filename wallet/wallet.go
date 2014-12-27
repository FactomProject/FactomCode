package wallet

import (
	"github.com/agl/ed25519"
	"crypto/rand"
)

//public / private key pair interface
/*
type ED25519_Key struct  {
	Pub  *[ed25519.PublicKeySize]byte
	Priv *[ed25519.PrivateKeySize]byte
}
*/

// Verifyer objects can Verify signed messages  
type Verifyer interface {
	Verify(msg []byte) bool
}

// Signer object can Sign msg  
type Signer interface {
	Sign(msg []byte) Signature
}

// PrivateKey contains Public/Private key pair   
type PrivateKey struct {
	Key  *[ed25519.PrivateKeySize]byte	
	Pub PublicKey
}

// PublicKey contains only Public part of Public/Private key pair   
type PublicKey struct {
	Key  *[ed25519.PublicKeySize]byte	
}

//Signature has signed data and its corresponsing PublicKey 
type Signature struct {
	Pub PublicKey
	Sig *[ed25519.SignatureSize]byte
}

// Sign signs msg with PrivateKey and return Signature
func (pk PrivateKey) Sign(msg []byte) (sig Signature) {
	sig.Pub = pk.Pub
	sig.Sig = ed25519.Sign(pk.Key, msg)
	return
}

// Verify returns true iff sig is a valid signature of msg by PublicKey.
func (sig Signature) Verify(msg []byte) bool {
	return ed25519.Verify(sig.Pub.Key, msg, sig.Sig)
}

//Generate creates new PrivateKey / PublciKey pair or returns error
func (pk *PrivateKey) GenerateKey() (err error) {
	pk.Pub.Key, pk.Key, err = ed25519.GenerateKey(rand.Reader)
	return err
}


