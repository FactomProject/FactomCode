package wallet


import (
	"github.com/agl/ed25519"
	"crypto/rand"
)

//public / private key pair interface
type ED25519_Key struct  {
	Pub  *[ed25519.PublicKeySize]byte
	Priv *[ed25519.PrivateKeySize]byte
}

// Sign signs the message with privateKey and returns a signature.
func (k *ED25519_Key) Sign(data []byte) (*[ed25519.SignatureSize]byte) {
	return ed25519.Sign(k.Priv, data)
}

// GenerateKey generates a public/private key pair using randomness from rand.
func NewED25519_Key() (k *ED25519_Key, err error) {
	k = new(ED25519_Key)
	k.Pub, k.Priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil { 
		return nil, err
	}

	return
}


// Verify returns true iff sig is a valid signature of message by publicKey.
func Verify(publicKey *[ed25519.PublicKeySize]byte, message []byte, sig *[ed25519.SignatureSize]byte) bool {
	return ed25519.Verify(publicKey, message, sig)
}