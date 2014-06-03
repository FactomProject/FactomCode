package data

import (
	"hash"
	"crypto/rsa"
	"crypto/ecdsa"
	"crypto/elliptic"
)

const (
	BadKeyType		= -1
	ECDSAPubKeyType	=  0
	ECDSAPrivKeyType=  1
	RSAPubKeyType	=  2
	RSAPrivKeyType	=  3
)

type Key struct {
	KeyType			int8
}

type ECDSAPubKey struct {
	Key
	ecdsa.PublicKey
}

type ECDSAPrivKey struct {
	Key
	ecdsa.PrivateKey
}

type RSAPubKey struct {
	Key
	rsa.PublicKey
}

type RSAPrivKey struct {
	Key
	rsa.PrivateKey
}

func (k *Key) writeToHash(h hash.Hash) (err error) {
	return nil
}

func (k *ECDSAPubKey) writeToHash(h hash.Hash) (err error) {
	if _, err = h.Write(elliptic.Marshal(k, k.X, k.Y)); err != nil {
		return err
	}
	
	if _, err = h.Write(k.X.Bytes()); err != nil {
		return err
	}
	
	_, err = h.Write(k.Y.Bytes())
	return err
}

func (k *ECDSAPrivKey) writeToHash(h hash.Hash) (err error) {
	pub := &ECDSAPubKey{k.Key, k.PrivateKey.PublicKey}
	if err = pub.writeToHash(h); err != nil {
		return err
	}
	
	_, err = h.Write(k.D.Bytes())
	return err
}