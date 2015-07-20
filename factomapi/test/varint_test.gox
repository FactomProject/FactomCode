// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package factomapi

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestReadVarint(t *testing.T) {
	var (
		num1 uint64 = 1e9
		num2 uint64 = 5
		num3 uint64 = 1099511627775
	)

	buf := new(bytes.Buffer)

	n, err := WriteVarint(buf, num1)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log("n =", n)
	t.Log("buf =", hex.EncodeToString(buf.Bytes()))

	n, err = WriteVarint(buf, num2)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log("n =", n)
	t.Log("buf =", hex.EncodeToString(buf.Bytes()))

	n, err = WriteVarint(buf, num3)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log("n =", n)
	t.Log("buf =", hex.EncodeToString(buf.Bytes()))

	x, err := ReadVarint(buf)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log("x =", x)
	t.Log("buf =", hex.EncodeToString(buf.Bytes()))

	x, err = ReadVarint(buf)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log("x =", x)
	t.Log("buf =", hex.EncodeToString(buf.Bytes()))

	x, err = ReadVarint(buf)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Log("x =", x)
	t.Log("buf =", hex.EncodeToString(buf.Bytes()))
}
