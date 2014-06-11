package notarydata

import (
	"bytes"
	"errors"
	"hash"
	
	"crypto/rsa"
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
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

func bigIntMarshalBinary(i *big.Int) (data []byte, err error) {
	intd, err := i.GobEncode()
	if err != nil { return }
	
	size := len(intd)
	if size > 255 { return nil, errors.New("Big int too big") }
	
	data = make([]byte, size)
	data[0] = byte(size)
	copy(data[1:], intd)
	return
}

func bigIntMarshalledSize(i *big.Int) uint64 {
	intd, err := i.GobEncode()
	if err != nil { return 0 }
	
	return uint64(len(intd))
}

func bigIntUnmarshalBinary(data []byte) (retd []byte, i *big.Int, err error) {
	size := uint8(data[0])
	
	i = new(big.Int)
	err = i.GobDecode(data[1:size+1])
	retd = data[size+1:]
	
	return
}

func (k *Key) MarshalBinary() (data []byte, err error) {
	return []byte{byte(k.KeyType)}, nil
}

func (k *ECDSAPubKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	data, err = k.Key.MarshalBinary()
	if err != nil { return }
	buf.Write(data)
	
	p := k.Params()
	
	data, err = bigIntMarshalBinary(p.P)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(p.N)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(p.B)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(p.Gx)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(p.Gy)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(k.X)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(k.Y)
	if err != nil { return }
	buf.Write(data)
	
	return buf.Bytes(), nil
}

func (k *ECDSAPrivKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	data, err = (&ECDSAPubKey{k.Key, k.PublicKey}).MarshalBinary()
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(k.D)
	if err != nil { return }
	buf.Write(data)
	
	return buf.Bytes(), nil
}

func (k *Key) MarshalledSize() uint64 {
	return 1 // KeyType int8
}

func (k *ECDSAPubKey) MarshalledSize() uint64 {
	s := k.Key.MarshalledSize()
	
	p := k.Params()
	s += bigIntMarshalledSize(p.P)
	s += bigIntMarshalledSize(p.N)
	s += bigIntMarshalledSize(p.B)
	s += bigIntMarshalledSize(p.Gx)
	s += bigIntMarshalledSize(p.Gy)
	
	s += bigIntMarshalledSize(k.X)
	s += bigIntMarshalledSize(k.Y)
	
	return s
}

func (k *ECDSAPrivKey) MarshalledSize() uint64 {
	s := (&ECDSAPubKey{k.Key, k.PublicKey}).MarshalledSize()
	
	s += bigIntMarshalledSize(k.D)
	
	return s
}

func (k *Key) UnmarshalBinary(data []byte) error {
	k.KeyType = int8(data[0])
	
	return nil
}

func (k *ECDSAPubKey) UnmarshalBinary(data []byte) (err error) {
	err = k.Key.UnmarshalBinary(data)
	if err != nil { return }
	data = data[1:]
	
	p := k.Params()
	
	data, p.P, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, p.N, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, p.B, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, p.Gx, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, p.Gy, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, k.X, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, k.Y, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	return
}

func (k *ECDSAPrivKey) UnmarshalBinary(data []byte) (err error) {
	pub := new(ECDSAPubKey)
	err = pub.UnmarshalBinary(data)
	if err != nil { return }
	
	data = data[pub.MarshalledSize():]
	
	k.KeyType = pub.KeyType
	k.PublicKey = pub.PublicKey
	
	data, k.D, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	return
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