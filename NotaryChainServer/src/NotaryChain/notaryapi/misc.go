package notaryapi

import (
	"encoding"
	"errors"
	
	"math/big"
)

type BinaryMarshallable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	MarshalledSize() uint64
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

type simpleData struct {
	data []byte
}

func (d *simpleData) MarshalBinary() ([]byte, error) {
	return d.data, nil
}

func (d *simpleData) MarshalledSize() uint64 {
	return uint64(len(d.data))
}

func (d *simpleData) UnmarshalBinary([]byte) error {
	return errors.New("simpleData cannot be unmarshalled")
}

