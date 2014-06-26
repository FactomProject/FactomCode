package notaryapi

import (
	"bytes"
	"errors"
	"reflect"
	
	"math/big"
	
	"github.com/firelizzard18/gocoding"
)

type Signature interface {
	Key
	HashMethod() string
	Key() Key
}

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
	R, S			*big.Int
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
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(s.R)
	if err != nil { return }
	buf.Write(data)
	
	data, err = bigIntMarshalBinary(s.S)
	if err != nil { return }
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
	if err != nil { return }
	data = data[s.ECDSAPubKey.MarshalledSize():]
	
	data, s.R, err = bigIntUnmarshalBinary(data)
	if err != nil { return }
	
	data, s.S, err = bigIntUnmarshalBinary(data)
	return
}

func (e *ECDSASignature) Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*ECDSASignature)
		
		renderer.StartStruct()
		
		renderer.StartElement(`Key`)
		marshaller.MarshalObject(renderer, &e.ECDSAPubKey)
		renderer.StopElement(`Key`)
		
		renderer.StartElement(`HashMethod`)
		marshaller.MarshalObject(renderer, e.HashMethod())
		renderer.StopElement(`HashMethod`)
		
		renderer.StartElement(`R`)
		marshaller.MarshalObject(renderer, e.R)
		renderer.StopElement(`R`)
		
		renderer.StartElement(`S`)
		marshaller.MarshalObject(renderer, e.S)
		renderer.StopElement(`S`)
		
		renderer.StopStruct()
	}
}