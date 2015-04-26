package common

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"math/big"
	//"fmt"
)

type BinaryMarshallable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}


type SimpleData struct {
	Data []byte
}

func (d *SimpleData) MarshalBinary() ([]byte, error) {
	return d.Data, nil
}

func (d *SimpleData) UnmarshalBinary([]byte) error {
	return errors.New("SimpleData cannot be unmarshalled")
}
