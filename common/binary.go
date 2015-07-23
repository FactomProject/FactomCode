package common

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

type BinaryMarshallable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

func bigIntMarshalBinary(i *big.Int) (data []byte, err error) {
	intd, err := i.GobEncode()
	if err != nil {
		return
	}

	size := len(intd)
	if size > 255 {
		return nil, errors.New("Big int is too big")
	}

	data = make([]byte, size+1)
	data[0] = byte(size)
	copy(data[1:], intd)
	return
}

func bigIntMarshalledSize(i *big.Int) uint64 {
	intd, err := i.GobEncode()
	if err != nil {
		return 0
	}

	return uint64(1 + len(intd))
}

func bigIntUnmarshalBinary(data []byte) (retd []byte, i *big.Int, err error) {
	size, data := uint8(data[0]), data[1:]

	i = new(big.Int)
	err, retd = i.GobDecode(data[:size]), data[size:]

	return
}

type SimpleData struct {
	Data []byte
}

func (d *SimpleData) MarshalBinary() ([]byte, error) {
	return d.Data, nil
}

func (d *SimpleData) MarshalledSize() uint64 {
	return uint64(len(d.Data))
}

func (d *SimpleData) UnmarshalBinary([]byte) error {
	return errors.New("SimpleData cannot be unmarshalled")
}

type ByteArray []byte

func (ba ByteArray) Bytes() []byte {
	newArray := make([]byte, len(ba))
	copy(newArray, ba[:])
	return newArray
}

func (ba ByteArray) SetBytes(newArray []byte) error {
	copy(ba[:], newArray[:])
	return nil
}

func (ba ByteArray) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	//fmt.Println("uint64(len(ba) ",uint64(len(ba)))

	binary.Write(&buf, binary.BigEndian, uint64(len(ba)))
	buf.Write(ba)
	return buf.Bytes(), nil
}

func (ba ByteArray) MarshalledSize() uint64 {
	//fmt.Println("uint64(len(ba) + 8)",uint64(len(ba) + 8))
	return uint64(len(ba) + 8)
}

func (ba ByteArray) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()
	newData = data
	count := binary.BigEndian.Uint64(newData[0:8])

	newData = newData[8:]

	tmp := make([]byte, count)

	copy(tmp[:], newData[:count])
	newData = newData[count:]

	return
}

func (ba ByteArray) UnmarshalBinary(data []byte) (err error) {
	_, err = ba.UnmarshalBinaryData(data)
	return
}

func NewByteArray(newHash []byte) (*ByteArray, error) {
	var sh ByteArray
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}
