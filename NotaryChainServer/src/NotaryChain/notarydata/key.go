package notarydata

import (
	"hash"
	"crypto/rsa"
	"crypto/ecdsa"
	"crypto/elliptic"
)

const (
	ECDSAPubKeyType	=  0
	ECDSAPrivKeyType=  1
	RSAPubKeyType	=  2
	RSAPrivKeyType	=  3
)

type Key struct {
	KeyType			int8			`json:"keyType"`
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

/*func (k *Key) UnmarshalBinary(data []byte) error {
	s.KeyType, data = int8(data[0]), data[1:]
	
	return nil
}

func (k *ECDSAPubKey) unmarshalBinary(data []byte) error {
	k.P, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.N, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.B, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.Gx, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.Gy, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.P, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.X, err := bigIntUnmarshal(data)
	if err != nil { return }
	
	k.Y, err := bigIntUnmarshal(data)
	if err != nil { return }
}

func (k *ECDSAPrivKey) unmarshalBinary(data []byte) error {
	ECDSAPubKey{-1, k.PrivateKey.PublicKey}.unmarshalBinary(data)
}

func (k *RSAPubKey) unmarshalBinary(data []byte) error {
	
}

func (k *RSAPrivKey) unmarshalBinary(data []byte) error {
	
}

/*func (k *Key) writeToHash(h hash.Hash) (err error) {
	return nil
}*/

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