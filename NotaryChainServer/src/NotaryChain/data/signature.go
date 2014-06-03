package data

import (
	"hash"
	"math/big"
)

type Signature struct {
	PublicKey		*Key
}

type ECDSASignature struct {
	Signature
	R, S			*big.Int
}

type RSASignature struct {
	Signature
	S				[]byte
}

func (s *Signature) writeToHash(h hash.Hash) (err error) {
	err = s.PublicKey.writeToHash(h)
	return err
}

func (s *ECDSASignature) writeToHash(h hash.Hash) (err error) {
	if err = s.Signature.writeToHash(h); err != nil {
		return err
	}
	
	if _, err = h.Write(s.R.Bytes()); err != nil {
		return err
	}
	
	_, err = h.Write(s.S.Bytes())
	return err
}

func (s *RSASignature) writeToHash(h hash.Hash) (err error) {
	if err = s.Signature.writeToHash(h); err != nil {
		return err
	}
	
	_, err = h.Write(s.S)
	return err
}