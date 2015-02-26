package notaryapi

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/agl/ed25519"
)

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
	Key *[ed25519.PrivateKeySize]byte
	Pub PublicKey
}

func (pk PrivateKey) Public() []byte {
	return (*pk.Pub.Key)[:]
}

func (pk *PrivateKey) AllocateNew() {
	pk.Key = new([64]byte)
	pk.Pub.Key = new([32]byte)
}

// PublicKey contains only Public part of Public/Private key pair
type PublicKey struct {
	Key *[ed25519.PublicKeySize]byte
}

func (pk PublicKey) String() string {
	return hex.EncodeToString((*pk.Key)[:])
}

func PubKeyFromString(instr string) (pk PublicKey) {
	p, _ := hex.DecodeString(instr)
	pk.Key = new([32]byte)
	copy(pk.Key[:], p)
	return
}

// Sign signs msg with PrivateKey and return Signature
func (pk PrivateKey) Sign(msg []byte) (sig Signature) {
	sig.Pub = pk.Pub
	sig.Sig = ed25519.Sign(pk.Key, msg)
	return
}

// Sign signs msg with PrivateKey and return Signature
func (pk PrivateKey) MarshalSign(msg BinaryMarshallable) (sig Signature) {
	data, _ := msg.MarshalBinary()
	return pk.Sign(data)
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

func (k PublicKey) Verify(msg []byte, sig *[ed25519.SignatureSize]byte) bool {
	return ed25519.Verify(k.Key, msg, sig)
}

// Verify returns true iff sig is a valid signature of message by publicKey.
func Verify(publicKey *[ed25519.PublicKeySize]byte, message []byte, sig *[ed25519.SignatureSize]byte) bool {
	return ed25519.Verify(publicKey, message, sig)
}

// Verify returns true iff sig is a valid signature of message by publicKey.
func VerifySlice(p []byte, message []byte, s []byte) bool {

	sig := new([64]byte)
	pub := new([32]byte)
	copy(sig[:], s)
	copy(pub[:], p)
	return Verify(pub, message, sig)
}
