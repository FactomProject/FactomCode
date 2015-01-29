package notaryapi

import (
	"bytes"
	"errors"
	"math/big"
	"reflect"
	"strings"

	"github.com/FactomProject/gocoding"
)

type Signature interface {
	Key
	HashMethod() string
	Key() Key
}

const (
	BadSignatureType   = 0
	ECDSASignatureType = 1
	RSASignatureType   = 2
)

func UnmarshalBinarySignature(data []byte) (s Signature, err error) {
	switch int(data[0]) {
	case ECDSAPubKeyType:
		s = new(ECDSASignature)

	default:
		return nil, errors.New("Bad signature type")
	}

	err = s.UnmarshalBinary(data)
	return
}

type ECDSASignature struct {
	ECDSAPubKey
	R, S *big.Int
}

func (s *ECDSASignature) HashMethod() string {
	return "SHA256"
}

func (s *ECDSASignature) Key() Key {
	return &s.ECDSAPubKey
}

func (s *ECDSASignature) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = s.ECDSAPubKey.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(s.R)
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = bigIntMarshalBinary(s.S)
	if err != nil {
		return
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

func (s *ECDSASignature) MarshalledSize() uint64 {
	size := s.ECDSAPubKey.MarshalledSize()

	size += bigIntMarshalledSize(s.R)
	size += bigIntMarshalledSize(s.S)

	return size
}

func (s *ECDSASignature) UnmarshalBinary(data []byte) (err error) {
	err = s.ECDSAPubKey.UnmarshalBinary(data)
	if err != nil {
		return
	}
	data = data[s.ECDSAPubKey.MarshalledSize():]

	data, s.R, err = bigIntUnmarshalBinary(data)
	if err != nil {
		return
	}

	data, s.S, err = bigIntUnmarshalBinary(data)
	return
}

func (e *ECDSASignature) Encoding(m gocoding.Marshaller, t reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, r gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*ECDSASignature)

		r.StartStruct()

		r.StartElement(`Type`)
		m.MarshalObject(r, ECDSASignatureType)
		r.StopElement(`Type`)

		r.StartElement(`Key`)
		m.MarshalObject(r, &e.ECDSAPubKey)
		r.StopElement(`Key`)

		r.StartElement(`HashMethod`)
		m.MarshalObject(r, e.HashMethod())
		r.StopElement(`HashMethod`)

		r.StartElement(`R`)
		m.MarshalObject(r, e.R)
		r.StopElement(`R`)

		r.StartElement(`S`)
		m.MarshalObject(r, e.S)
		r.StopElement(`S`)

		r.StopStruct()
	}
}

func (e *ECDSASignature) ElementDecoding(m gocoding.Unmarshaller,
	t reflect.Type) gocoding.Decoder {
	return func(scratch [64]byte, s gocoding.Scanner, value reflect.Value) {
		e := value.Interface().(*ECDSASignature)
		e.R = new(big.Int)
		e.S = new(big.Int)

		for {
			// get the next code, check for the end
			code := s.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) {
				break
			}

			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				gocoding.PeekCheck(s, gocoding.ScannedKeyBegin,
					gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
				return
			}

			// get the key
			key := s.NextValue()
			if key.Kind() != reflect.String {
				s.Error(gocoding.ErrorPrint("Decoding", "Invalid key type %s",
					key.Type().String()))
			}

			field := reflect.Value{}
			switch strings.ToLower(key.String()) {
			case `key`:
				field = reflect.ValueOf(&e.ECDSAPubKey)

			case `r`:
				field = reflect.ValueOf(&e.R)

			case `s`:
				field = reflect.ValueOf(&e.S)
			}

			s.Continue()
			if !field.IsValid() {
				s.NextValue()
			} else {
				m.UnmarshalValue(s, field)
			}
		}
	}
}
