// Decode a variable integer from the given data buffer.
// Returns the uint64 bit value and a data slice positioned
// after the variable integer

package common

import (
	"bytes"
)

func DecodeVarInt(data []byte) (uint64, []byte) {
	b := uint8(data[0])
	if b < 0xfd {
		return uint64(b), data[1:]
	}

	var v uint64

	v = (uint64(data[1]) << 8) | uint64(data[2])
	if b == 0xfd {
		return v, data[3:]
	}

	v = v << 16
	v = v | (uint64(data[3]) << 8) | uint64(data[4])

	if b == 0xfe {
		return v, data[5:]
	}

	v = v << 16
	v = v | (uint64(data[5]) << 8) | uint64(data[6])

	v = v << 16
	v = v | (uint64(data[7]) << 8) | uint64(data[8])

	return v, data[9:]
}

// Encode an integer as a variable int into the given data buffer.
func EncodeVarInt(out *bytes.Buffer, v uint64) error {
	var err error
	switch {
	case v < 0xfd:
		err = out.WriteByte(byte(v))
		if err != nil {
			return err
		}
	case v <= 0xFFFF:
		out.WriteByte(0xfd)
		err = out.WriteByte(byte(v >> 8))
		if err != nil {
			return err
		}
		err = out.WriteByte(byte(v))
		if err != nil {
			return err
		}
	case v <= 0xFFFFFFFF:
		out.WriteByte(0xfe)
		for i := 0; i < 4; i++ {
			v = (v>>24)&0xFF + v<<8
			err = out.WriteByte(byte(v))
			if err != nil {
				return err
			}
		}
	default:
		out.WriteByte(0xff)
		for i := 0; i < 8; i++ {
			v = (v>>56)&0xFF + v<<8
			err = out.WriteByte(byte(v))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ReadVarInt(buf *bytes.Buffer) uint64 {
	b, err := buf.ReadByte()
	if err != nil {
		return 0
	}
	if b < 0xfd {
		return uint64(b)
	}

	var v uint64
	
	if p := buf.Next(2); p != nil {
		v = (uint64(p[0]) << 8) | uint64(p[1])
	}
	if b == 0xfd {
		return v
	}

	v = v << 16
	if p := buf.Next(2); p != nil {
		v = v | (uint64(p[0]) << 8) | uint64(p[1])
	}
	if b == 0xfe {
		return v
	}

	v = v << 16
	if p := buf.Next(2); p != nil {
		v = v | (uint64(p[0]) << 8) | uint64(p[1])
	}
	v = v << 16
	if p := buf.Next(2); p != nil {
		v = v | (uint64(p[0]) << 8) | uint64(p[1])
	}
	return v
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
