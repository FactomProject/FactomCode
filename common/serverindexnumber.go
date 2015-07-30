// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
)

const (
	ServerIndexNumberSize = 1
)

type ServerIndexNumber struct {
	Printable
	BinaryMarshallable

	Number uint8
}

func NewServerIndexNumber() *ServerIndexNumber {
	return new(ServerIndexNumber)
}

func (s *ServerIndexNumber) ECID() byte {
	return ECIDServerIndexNumber
}

func (s *ServerIndexNumber) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte(s.Number)
	return buf.Bytes(), nil
}

func (s *ServerIndexNumber) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	var c byte
	if c, err = buf.ReadByte(); err != nil {
		return
	} else {
		s.Number = c
	}
	newData = buf.Bytes()
	return
}

func (s *ServerIndexNumber) UnmarshalBinary(data []byte) (err error) {
	_, err = s.UnmarshalBinaryData(data)
	return
}

func (e *ServerIndexNumber) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *ServerIndexNumber) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *ServerIndexNumber) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *ServerIndexNumber) Spew() string {
	return Spew(e)
}
