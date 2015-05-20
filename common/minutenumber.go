// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
)

type MinuteNumber struct {
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
	buf.WriteByte(m.ECID())
	buf.WriteByte(m.Number)
	return buf.Bytes(), nil
}

func (m *MinuteNumber) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if _, err := buf.ReadByte(); err != nil {
		return err
	}
	if c, err := buf.ReadByte(); err != nil {
		return err
	} else {
		m.Number = c
	}
	return nil
}
