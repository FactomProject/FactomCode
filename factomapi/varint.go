package factomapi

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

func ReadVarint(r byteAndReader) (x uint64, err error) {
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
	
	m := uint64(1)
	for i, j := 0, n-1; i <= j; i++ {
		x += uint64(val[j-i]) * m
		m *= 256
	}
	
	return
}

func WriteVarint(w io.Writer, x uint64) (int, error) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, x)
	val := buf.Bytes()
	p := make([]byte, 1, 9)
	
	switch {
	case x < 0xFD:
		p[0] = byte(x)
	case x <= 0xFFFF:
		p[0] = 0xFD
		p = append(p, val[6:]...)
	case x <= 0xFFFFFFFF:
		p[0] = 0xFE
		p = append(p, val[4:]...)
	case x <= 0xFFFFFFFFFFFFFFFF:
		p[0] = 0xFF
		p = append(p, val...)
	}
	
	return w.Write(p)
}
