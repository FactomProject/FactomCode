package notaryapi

import (
	"github.com/agl/ed25519"
)


//Signature has signed data and its corresponsing PublicKey 
type Signature struct {
	Pub PublicKey
	Sig *[ed25519.SignatureSize]byte
}

func (sig Signature) Key() []byte {
	return (*sig.Pub.Key)[:]
}

func UnmarshalBinarySignature(data []byte) (sig Signature) {
	sig.Pub.Key = new([32]byte)
	sig.Sig = new([64]byte) 
	copy(sig.Pub.Key[:],data[:32])
	data = data[32:]
	copy(sig.Sig[:],data[:64])
	return
}