package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type byteAndReader interface {
	io.Reader
	io.ByteReader
}

func ReadVarint(r byteAndReader) (uint64, error) {
	var blen int
	
	b, err := r.ReadByte()
	switch {
	case b < 0xFD:
		return uint64(b), nil
	case b == 0xFD:
		blen = 2
	case b == 0xFE:
		blen = 4
	case b == 0xFF:
		blen = 8
	}
	
	val := make([]byte, blen)
	n, err := r.Read(val)
	if err != nil {
		return 0, err
	}
	if n != len(val) {
		return 0, fmt.Errorf("Could not read Varint from %v", r)
	}
	
	x, n := binary.Uvarint(val)
	if n == 0 {
		return 0, fmt.Errorf("%v is too small\n", val)
	}
	if n < 0 {
		return 0, fmt.Errorf("%v is too large\n", val)
	}
	
	return x, nil
}

func WriteVarint(w io.Writer, x uint64) (int, error){
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, x)
	val := buf.Bytes()
	b := make([]byte, 1, 9)
	
	switch {
	case x < 0xFD:
		b[0] = byte(x)
	case x <= 0xFFFF:
		b[0] = 0xFD
		b = append(b, val[7:]...)
	case x <= 0xFFFFFFFF:
		b[0] = 0xFE
		b = append(b, val[5:]...)
	case x <= 0xFFFFFFFFFFFFFFFF:
		b[0] = 0xFF
		b = append(b, val...)
	}
	
	return w.Write(b)
}
