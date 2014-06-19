package notaryapi

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	
	"crypto/sha256"
	"crypto/ecdsa"
	
	"github.com/firelizzard18/gocoding"
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
	Key *ecdsa.PublicKey
}

func (k *ECDSAPubKey) KeyType() int8 {
	return ECDSAPubKeyType
}

func (k *ECDSAPubKey) Verify(data []byte, sig Signature) bool {
	if sig.HashMethod() != "SHA256" {
		return false
	}
	
	if sig.KeyType() != k.KeyType() {
		return false
	}
	
	esig := sig.(*ECDSASignature)
	
	hash := sha256.Sum256(data)
	return ecdsa.Verify(k.Key, hash[:], esig.R, esig.S)
}

func (k *ECDSAPubKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	buf.Write([]byte{byte(ECDSAPubKeyType)})
	
	p := k.Key.Params()
	
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
	
	data, err = bigIntMarshalBinary(k.Key.X)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(k.Key.Y)
	if err != nil { return }
	buf.Write(data)
	
	return buf.Bytes(), nil
}

func (k *ECDSAPubKey) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 1
	
	p := k.Key.Params()
	size += bigIntMarshalledSize(p.P)
	size += bigIntMarshalledSize(p.N)
	size += bigIntMarshalledSize(p.B)
	size += bigIntMarshalledSize(p.Gx)
	size += bigIntMarshalledSize(p.Gy)
	
	size += bigIntMarshalledSize(k.Key.X)
	size += bigIntMarshalledSize(k.Key.Y)
	
	return size
}

func (k *ECDSAPubKey) UnmarshalBinary(data []byte) (err error) {
	data = data[1:]
	
	p := k.Key.Params()
	
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
	
	data, k.Key.X, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, k.Key.Y, err = bigIntUnmarshalBinary(data)
	return
}

func (e *ECDSAPubKey) Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*ECDSAPubKey)
		
		renderer.StartStruct()
		
		renderer.StartElement(`X`)
		marshaller.MarshalObject(e.Key.X)
		renderer.StopElement(`X`)
		
		renderer.StartElement(`Y`)
		marshaller.MarshalObject(e.Key.Y)
		renderer.StopElement(`Y`)
		
		renderer.StartElement(`Curve`)
		marshaller.MarshalObject(e.Key.Params())
		renderer.StopElement(`Curve`)
		
		renderer.StopStruct()
	}
}

type ECDSAPrivKey struct {
	Key *ecdsa.PrivateKey
}

func (k *ECDSAPrivKey) KeyType() int8 {
	return ECDSAPrivKeyType
}

func (k *ECDSAPrivKey) Sign(rand io.Reader, data []byte) (Signature, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand, k.Key, hash[:])
	if err != nil { return nil, err }
	
	return &ECDSASignature{ECDSAPubKey{&k.Key.PublicKey}, r, s}, nil
}

func (k *ECDSAPrivKey) Verify(data []byte, sig Signature) bool {
	pub := &ECDSAPubKey{&k.Key.PublicKey}
	return pub.Verify(data, sig)
}

func (k *ECDSAPrivKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	data, err = (&ECDSAPubKey{&k.Key.PublicKey}).MarshalBinary()
	if err != nil { return }
	data[0] = byte(ECDSAPrivKeyType)
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(k.Key.D)
	if err != nil { return }
	buf.Write(data)
	
	return buf.Bytes(), nil
}

func (k *ECDSAPrivKey) MarshalledSize() uint64 {
	s := (&ECDSAPubKey{&k.Key.PublicKey}).MarshalledSize()
	
	s += bigIntMarshalledSize(k.Key.D)
	
	return s
}

func (k *ECDSAPrivKey) UnmarshalBinary(data []byte) (err error) {
	data = data[1:]
	
	pub := new(ECDSAPubKey)
	err = pub.UnmarshalBinary(data)
	if err != nil { return }
	
	data = data[pub.MarshalledSize():]
	
	k.Key.PublicKey = *pub.Key
	
	data, k.Key.D, err = bigIntUnmarshalBinary(data)
	return
}

func (e *ECDSAPrivKey) Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*ECDSAPrivKey)
		
		renderer.StartStruct()
		
		renderer.StartElement(`X`)
		marshaller.MarshalObject(e.Key.X)
		renderer.StopElement(`X`)
		
		renderer.StartElement(`Y`)
		marshaller.MarshalObject(e.Key.Y)
		renderer.StopElement(`Y`)
		
		renderer.StartElement(`D`)
		marshaller.MarshalObject(e.Key.D)
		renderer.StopElement(`D`)
		
		renderer.StartElement(`Curve`)
		marshaller.MarshalObject(e.Key.Params())
		renderer.StopElement(`Curve`)
		
		renderer.StopStruct()
	}
}