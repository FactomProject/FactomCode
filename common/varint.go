// Decode a variable integer from the given data buffer.
// Returns the uint64 bit value and a data slice positioned
// after the variable integer

package common

import (
	"bytes"
	"io"
)

func DecodeVarInt(data []byte) (uint64, []byte) {
	buf := bytes.NewBuffer(data)
	resp, err := ReadVarIntWithError(buf)
	if err != nil {
		panic(err)
	}
	return resp, buf.Bytes()
}

func VarIntLength(v uint64) int {
	switch {
	case v < 0xfd:
		return 1
	case v <= 0xFFFF:
		return 3
	case v <= 0xFFFFFFFF:
		return 5
	default:
		return 9
	}
	return -1
}

// Encode an integer as a variable int into the given data buffer.
func EncodeVarInt(out *bytes.Buffer, v uint64) error {
	_, err := WriteVarInt(out, v)
	return err
}

func ReadVarIntWithError(buf *bytes.Buffer) (uint64, error) {
	b, err := buf.ReadByte()
	if err != nil {
		return 0, err
	}
	if b < 0xfd {
		return uint64(b), nil
	}

	var v uint64

	if buf.Len() < 2 {
		return 0, io.EOF
	}

	if p := buf.Next(2); p != nil {
		v = (uint64(p[0]) << 8) | uint64(p[1])
	}
	if b == 0xfd {
		return v, nil
	}

	if buf.Len() < 2 {
		return 0, io.EOF
	}
	v = v << 16
	if p := buf.Next(2); p != nil {
		v = v | (uint64(p[0]) << 8) | uint64(p[1])
	}
	if b == 0xfe {
		return v, nil
	}

	if buf.Len() < 2 {
		return 0, io.EOF
	}
	v = v << 16
	if p := buf.Next(2); p != nil {
		v = v | (uint64(p[0]) << 8) | uint64(p[1])
	}
	if buf.Len() < 2 {
		return 0, io.EOF
	}
	v = v << 16
	if p := buf.Next(2); p != nil {
		v = v | (uint64(p[0]) << 8) | uint64(p[1])
	}
	return v, nil

}

func ReadVarInt(buf *bytes.Buffer) uint64 {
	u, _ := ReadVarIntWithError(buf)
	return u
}

func WriteVarInt(buf *bytes.Buffer, v uint64) (n int, err error) {
	switch {
	case v < 0xfd:
		err = buf.WriteByte(byte(v))
		if err != nil {
			return n, err
		}
		n += 1
	case v <= 0xFFFF:
		buf.WriteByte(0xfd)
		err = buf.WriteByte(byte(v >> 8))
		if err != nil {
			return n, err
		}
		n += 1
		err = buf.WriteByte(byte(v))
		if err != nil {
			return n, err
		}
		n += 1
	case v <= 0xFFFFFFFF:
		buf.WriteByte(0xfe)
		n += 1
		for i := 0; i < 4; i++ {
			v = (v>>24)&0xFF + v<<8
			err = buf.WriteByte(byte(v))
			if err != nil {
				return n, err
			}
			n += 1
		}
	default:
		buf.WriteByte(0xff)
		n += 1
		for i := 0; i < 8; i++ {
			v = (v>>56)&0xFF + v<<8
			err = buf.WriteByte(byte(v))
			if err != nil {
				return n, err
			}
			n += 1
		}
	}
	return n, nil
}
