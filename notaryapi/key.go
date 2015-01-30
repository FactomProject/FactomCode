package notaryapi

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	//"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"reflect"

	"github.com/FactomProject/gocoding"
)

const (
	BadKeyType       = -1
	ECDSAPubKeyType  = 0
	ECDSAPrivKeyType = 1
	RSAPubKeyType    = 2
	RSAPrivKeyType   = 3
)

type Key interface {
	BinaryMarshallable
	KeyType() int8
	Public() PublicKey
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

func GenerateKeyPair(keyType int8, rand io.Reader) (key PrivateKey, err error) {
	switch keyType {
	case ECDSAPrivKeyType:
		var _key *ecdsa.PrivateKey
		_key, err = ecdsa.GenerateKey(elliptic.P256(), rand)
		if err != nil {
			return
		}
		key = &ECDSAPrivKey{_key}
	}
	return
}

type ECDSAPubKey struct {
	Key *ecdsa.PublicKey
}

func (k *ECDSAPubKey) KeyType() int8 {
	return ECDSAPubKeyType
}

func (k *ECDSAPubKey) Public() PublicKey {
	return k
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
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(p.N)
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(p.B)
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(p.Gx)
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(p.Gy)
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, int64(p.BitSize))

	data, err = bigIntMarshalBinary(k.Key.X)
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(k.Key.Y)
	if err != nil {
		return
	}
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
	size += 8 // p.BitSize int64
	size += bigIntMarshalledSize(k.Key.X)
	size += bigIntMarshalledSize(k.Key.Y)

	return size
}

func (k *ECDSAPubKey) UnmarshalBinary(data []byte) (err error) {
	data = data[1:]

	k.Key = new(ecdsa.PublicKey)
	k.Key.Curve = new(elliptic.CurveParams)

	p := k.Key.Params()

	data, p.P, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	data, p.N, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	data, p.B, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	data, p.Gx, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	data, p.Gy, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	p.BitSize, data = int(binary.BigEndian.Uint64(data[:8])), data[8:]

	data, k.Key.X, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	data, k.Key.Y, err = bigIntUnmarshalBinary(data)
	return
}

func (e *ECDSAPubKey) Encoding(m gocoding.Marshaller,
	t reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, r gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*ECDSAPubKey)

		r.StartStruct()

		r.StartElement(`X`)
		m.MarshalObject(r, e.Key.X)
		r.StopElement(`X`)

		r.StartElement(`Y`)
		m.MarshalObject(r, e.Key.Y)
		r.StopElement(`Y`)

		r.StartElement(`Curve`)
		m.MarshalObject(r, e.Key.Params())
		r.StopElement(`Curve`)

		r.StopStruct()
	}
}

func (e *ECDSAPubKey) DecodableFields() map[string]reflect.Value {
	e.Key = new(ecdsa.PublicKey)
	e.Key.X = new(big.Int)
	e.Key.Y = new(big.Int)
	p := new(elliptic.CurveParams)
	e.Key.Curve = p
	p.B = new(big.Int)
	p.Gx = new(big.Int)
	p.Gy = new(big.Int)
	p.N = new(big.Int)
	p.P = new(big.Int)

	fields := map[string]reflect.Value{
		`X`:     reflect.ValueOf(e.Key.X),
		`Y`:     reflect.ValueOf(e.Key.Y),
		`Curve`: reflect.ValueOf(e.Key.Curve),
	}

	return fields
}

type ECDSAPrivKey struct {
	Key *ecdsa.PrivateKey
}

func (k *ECDSAPrivKey) KeyType() int8 {
	return ECDSAPrivKeyType
}

func (k *ECDSAPrivKey) public() ECDSAPubKey {
	return ECDSAPubKey{&k.Key.PublicKey}
}

func (k *ECDSAPrivKey) Public() PublicKey {
	pub := k.public()
	return &pub
}

func (k *ECDSAPrivKey) Sign(rand io.Reader, data []byte) (Signature, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand, k.Key, hash[:])
	if err != nil {
		return nil, err
	}
	return &ECDSASignature{k.public(), r, s}, nil
}

func (k *ECDSAPrivKey) Verify(data []byte, sig Signature) bool {
	return k.Public().Verify(data, sig)
}

func (k *ECDSAPrivKey) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = k.Public().MarshalBinary()
	if err != nil {
		return
	}
	data[0] = byte(ECDSAPrivKeyType)
	buf.Write(data)

	data, err = bigIntMarshalBinary(k.Key.D)
	if err != nil {
		return
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

func (k *ECDSAPrivKey) MarshalledSize() uint64 {
	s := k.Public().MarshalledSize()
	s += bigIntMarshalledSize(k.Key.D)
	return s
}

func (k *ECDSAPrivKey) UnmarshalBinary(data []byte) (err error) {
	pub := new(ECDSAPubKey)
	err = pub.UnmarshalBinary(data)
	if err != nil {
		return
	}

	data = data[pub.MarshalledSize():]

	k.Key = new(ecdsa.PrivateKey)

	k.Key.PublicKey = *pub.Key

	data, k.Key.D, err = bigIntUnmarshalBinary(data)
	return
}

func (e *ECDSAPrivKey) Encoding(m gocoding.Marshaller,
	t reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, r gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*ECDSAPrivKey)

		r.StartStruct()

		r.StartElement(`X`)
		m.MarshalObject(r, e.Key.X)
		r.StopElement(`X`)

		r.StartElement(`Y`)
		m.MarshalObject(r, e.Key.Y)
		r.StopElement(`Y`)

		r.StartElement(`D`)
		m.MarshalObject(r, e.Key.D)
		r.StopElement(`D`)

		r.StartElement(`Curve`)
		m.MarshalObject(r, e.Key.Params())
		r.StopElement(`Curve`)

		r.StopStruct()
	}
}

func (e *ECDSAPrivKey) DecodableFields() map[string]reflect.Value {
	e.Key = new(ecdsa.PrivateKey)
	e.Key.X = new(big.Int)
	e.Key.Y = new(big.Int)
	e.Key.D = new(big.Int)
	p := new(elliptic.CurveParams)
	e.Key.Curve = p
	p.B = new(big.Int)
	p.Gx = new(big.Int)
	p.Gy = new(big.Int)
	p.N = new(big.Int)
	p.P = new(big.Int)

	fields := map[string]reflect.Value{
		`X`:     reflect.ValueOf(e.Key.X),
		`Y`:     reflect.ValueOf(e.Key.Y),
		`D`:     reflect.ValueOf(e.Key.D),
		`Curve`: reflect.ValueOf(e.Key.Curve),
	}

	return fields
}
