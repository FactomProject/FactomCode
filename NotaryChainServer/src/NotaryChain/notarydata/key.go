package notarydata

import (
	"bytes"
	"errors"
	"io"
	
	"crypto/ecdsa"
)

const (
	BadKeyType		= -1
	ECDSAPubKeyType	=  0
	ECDSAPrivKeyType=  1
	RSAPubKeyType	=  2
	RSAPrivKeyType	=  3
)

type Key interface {
	BinaryMarshallable
	KeyType()		int8
}

type PublicKey interface {
	Key
	Verify(data []byte, sig Signature) bool
}

type PrivateKey interface {
	PublicKey
	Sign(rand io.Reader, data []byte) (Signature, error)
}

func KeyTypeName(keyType int8) string {
	switch keyType {
	case ECDSAPubKeyType:
		return "ECDSA Public"
		
	case ECDSAPrivKeyType:
		return "ECDSA Private"
		
	case RSAPubKeyType:
		return "RSA Public"
		
	case RSAPrivKeyType:
		return "RSA Private"
		
	default:
		return "Unknown"
	}
}

func KeyTypeCode(keyType string) int8 {
	switch keyType {
	case "ECDSA Public":
		return ECDSAPubKeyType
		
	case "ECDSA Private":
		return ECDSAPrivKeyType
		
	case "RSA Public":
		return RSAPubKeyType
		
	case "RSA Private":
		return RSAPrivKeyType
		
	default:
		return BadKeyType
	}
}

func UnmarshalBinaryKey(data []byte) (k Key, err error) {
	switch int(data[0]) {
	case ECDSAPubKeyType:
		k = new(ECDSAPubKey)
		
	case ECDSAPrivKeyType:
		k = new(ECDSAPrivKey)
		
	default:
		return nil, errors.New("Bad key type")
	}
	
	err = k.UnmarshalBinary(data)
	return
}

type ECDSAPubKey struct {
	ecdsa.PublicKey
}

func (k *ECDSAPubKey) KeyType() int8 {
	return ECDSAPubKeyType
}

func (k *ECDSAPubKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	buf.Write([]byte{byte(ECDSAPubKeyType)})
	
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

func (k *ECDSAPubKey) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 1
	
	p := k.Params()
	size += bigIntMarshalledSize(p.P)
	size += bigIntMarshalledSize(p.N)
	size += bigIntMarshalledSize(p.B)
	size += bigIntMarshalledSize(p.Gx)
	size += bigIntMarshalledSize(p.Gy)
	
	size += bigIntMarshalledSize(k.X)
	size += bigIntMarshalledSize(k.Y)
	
	return size
}

func (k *ECDSAPubKey) UnmarshalBinary(data []byte) (err error) {
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
	return
}

type ECDSAPrivKey struct {
	ecdsa.PrivateKey
}

func (k *ECDSAPrivKey) KeyType() int8 {
	return ECDSAPrivKeyType
}

func (k *ECDSAPrivKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	data, err = (&ECDSAPubKey{k.PublicKey}).MarshalBinary()
	if err != nil { return }
	data[0] = byte(ECDSAPrivKeyType)
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(k.D)
	if err != nil { return }
	buf.Write(data)
	
	return buf.Bytes(), nil
}

func (k *ECDSAPrivKey) MarshalledSize() uint64 {
	s := (&ECDSAPubKey{k.PublicKey}).MarshalledSize()
	
	s += bigIntMarshalledSize(k.D)
	
	return s
}

func (k *ECDSAPrivKey) UnmarshalBinary(data []byte) (err error) {
	data = data[1:]
	
	pub := new(ECDSAPubKey)
	err = pub.UnmarshalBinary(data)
	if err != nil { return }
	
	data = data[pub.MarshalledSize():]
	
	k.PublicKey = pub.PublicKey
	
	data, k.D, err = bigIntUnmarshalBinary(data)
	return
}