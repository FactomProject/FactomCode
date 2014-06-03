package data

import (
	"hash"
)

const (
	BadKeyType		= -1
	ECCKeyType		=  0
	RSAKeyType		=  1
)

type Key struct {
	KeyType			int8
	KeyData			[]byte
}

func (k *Key) writeToHash(h hash.Hash) (err error) {
	_, err = h.Write(k.KeyData)
	return err
}