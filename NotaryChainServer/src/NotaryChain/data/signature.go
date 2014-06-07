package data

import (
	"hash"
)

type Signature struct {
	PublicKey		*Key
	SignedHash		*Hash
}

func (s *Signature) writeToHash(h hash.Hash) (err error) {
	if err = s.PublicKey.writeToHash(h); err != nil {
		return err
	}
	
	err = s.SignedHash.writeToHash(h)
	return err
} 