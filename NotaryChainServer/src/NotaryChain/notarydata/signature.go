package notarydata

import (
	"bytes"
	"errors"
	
	"math/big"
)

type Signature interface {
	Key
}

func UnmarshalBinarySignature(data []byte) (s Signature, err error) {
	switch int(data[0]) {
	case ECDSAPubKeyType:
		s = new(ECDSASignature)
		
	default:
		return nil, errors.New("Bad key type")
	}
	
	err = s.UnmarshalBinary(data)
	return
}

type ECDSASignature struct {
	ECDSAPubKey
	R, S			*big.Int
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