package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestWriteVarint(t *testing.T) {
	var (
		bigNum uint64 = 0xFFFFF
		smallNum uint64 = 5
	)
	
	buf := new(bytes.Buffer)

	n, err := WriteVarint(buf, bigNum)
	if err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("n=", n)
	fmt.Println(hex.EncodeToString(buf.Bytes()))

	n, err = WriteVarint(buf, smallNum)
	if err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("n=", n)
	fmt.Println(hex.EncodeToString(buf.Bytes()))
}

func TestReadVarint(t *testing.T) {
	var num uint64 = 0xFFFFFFFFF
	buf := new(bytes.Buffer)

	n, err := WriteVarint(buf, num)
	if err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("n =", n)
	fmt.Println("buf =", hex.EncodeToString(buf.Bytes()))
	
	x, err := ReadVarint(buf)
	if err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("x =", x)
	fmt.Println("buf =", hex.EncodeToString(buf.Bytes()))
}
