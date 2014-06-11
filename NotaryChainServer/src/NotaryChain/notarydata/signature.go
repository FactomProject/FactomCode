package notarydata

import (
	"hash"
	"math/big"
)

const (
	ECDSAKeyType = 1
	RSAKeyType = 2
)

type Signature struct {
	KeyType			int8
	PublicKey		*Key			`json:"key"`
}

type ECDSASignature struct {
	Signature
	R, S			*big.Int
}

type RSASignature struct {
	Signature
	S				[]byte
}

func (s *Signature) MarshalBinary() (data []byte, err error) {
	return
}

func (s *Signature) MarshalledSize() uint64 {
	return 0
}

func (s *Signature) UnmarshalBinary(data []byte) error {
	//s.KeyType = 
	
	return nil
}

func (s *ECDSASignature) UnmarshalBinary(data []byte) error {
	
	
	return nil
}

func (s *RSASignature) UnmarshalBinary(data []byte) error {
	
	
	return nil
}

func (s *Signature) writeToHash(h hash.Hash) (err error) {
	//err = s.PublicKey.writeToHash(h)
	return err
}

func (s *ECDSASignature) writeToHash(h hash.Hash) (err error) {
	if err = s.Signature.writeToHash(h); err != nil {
		return err
	}
	
	if _, err = h.Write(s.R.Bytes()); err != nil {
		return err
	}
	
	_, err = h.Write(s.S.Bytes())
	return err
}

func (s *RSASignature) writeToHash(h hash.Hash) (err error) {
	if err = s.Signature.writeToHash(h); err != nil {
		return err
	}
	
	_, err = h.Write(s.S)
	return err
}