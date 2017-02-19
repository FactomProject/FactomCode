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

	UnmarshalBinaryData([]byte) ([]byte, error)
	MarshalledSize() uint64
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

type ByteSlice32 [32]byte

var _ Printable = (*ByteSlice32)(nil)
var _ BinaryMarshallable = (*ByteSlice32)(nil)

func (bs ByteSlice32) MarshalBinary() ([]byte, error) {
	return bs[:], nil
}

func (bs ByteSlice32) MarshalledSize() uint64 {
	return 32
}

func (bs ByteSlice32) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()
	copy(bs[:], data[:32])
	newData = data[:32]
	return
}

func (bs ByteSlice32) UnmarshalBinary(data []byte) (err error) {
	copy(bs[:], data[:32])
	return
}

func (e *ByteSlice32) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ByteSlice32) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ByteSlice32) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ByteSlice32) Spew() string {
	return Spew(e)
}

func (bs *ByteSlice32) String() string {
	return fmt.Sprintf("%x", bs[:])
}

func (bs ByteSlice32) MarshalText() ([]byte, error) {
	return []byte(bs.String()), nil
}

type ByteSlice64 [64]byte

var _ Printable = (*ByteSlice64)(nil)
var _ BinaryMarshallable = (*ByteSlice64)(nil)

func (bs ByteSlice64) MarshalBinary() ([]byte, error) {
	return bs[:], nil
}

func (bs ByteSlice64) MarshalledSize() uint64 {
	return 64
}

func (bs ByteSlice64) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()
	copy(bs[:], data[:64])
	newData = data[:64]
	return
}

func (bs ByteSlice64) UnmarshalBinary(data []byte) (err error) {
	copy(bs[:], data[:64])
	return
}

func (e *ByteSlice64) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ByteSlice64) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ByteSlice64) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ByteSlice64) Spew() string {
	return Spew(e)
}

func (bs *ByteSlice64) String() string {
	return fmt.Sprintf("%x", bs[:])
}

func (bs ByteSlice64) MarshalText() ([]byte, error) {
	return []byte(bs.String()), nil
}

type ByteSlice6 [6]byte

var _ Printable = (*ByteSlice6)(nil)
var _ BinaryMarshallable = (*ByteSlice6)(nil)

func (bs ByteSlice6) MarshalBinary() ([]byte, error) {
	return bs[:], nil
}

func (bs ByteSlice6) MarshalledSize() uint64 {
	return 6
}

func (bs ByteSlice6) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()
	copy(bs[:], data[:6])
	newData = data[:6]
	return
}

func (bs ByteSlice6) UnmarshalBinary(data []byte) (err error) {
	copy(bs[:], data[:6])
	return
}

func (e *ByteSlice6) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ByteSlice6) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ByteSlice6) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ByteSlice6) Spew() string {
	return Spew(e)
}

func (bs *ByteSlice6) String() string {
	return fmt.Sprintf("%x", bs[:])
}

func (bs ByteSlice6) MarshalText() ([]byte, error) {
	return []byte(bs.String()), nil
}
