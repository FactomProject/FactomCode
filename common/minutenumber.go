// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
)

const (
	MinuteNumberSize = 1
)

type MinuteNumber struct {
	Printable          `json:"-"`
	BinaryMarshallable `json:"-"`

	Number uint8
}

func NewMinuteNumber() *MinuteNumber {
	return new(MinuteNumber)
}

func (m *MinuteNumber) ECID() byte {
	return ECIDMinuteNumber
}

func (m *MinuteNumber) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte(m.Number)
	return buf.Bytes(), nil
}

func (m *MinuteNumber) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	buf := bytes.NewBuffer(data)
	var c byte
	if c, err = buf.ReadByte(); err != nil {
		return
	} else {
		m.Number = c
	}
	newData = buf.Bytes()
	return
}

func (m *MinuteNumber) UnmarshalBinary(data []byte) (err error) {
	_, err = m.UnmarshalBinaryData(data)
	return
}

func (e *MinuteNumber) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *MinuteNumber) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *MinuteNumber) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *MinuteNumber) Spew() string {
	return Spew(e)
}
