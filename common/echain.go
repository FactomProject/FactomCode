// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"sync"
)

type EChain struct {
	ChainID         *Hash
	FirstEntry      *Entry
	NextBlock       *EBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

func NewEChain() *EChain {
	e := new(EChain)
	e.ChainID = NewHash()
	e.FirstEntry = NewEntry()
	e.NextBlock = NewEBlock()
	return e
}

func (e *EChain) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(e.ChainID.Bytes())

	if p, err := e.FirstEntry.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func (e *EChain) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	newData = data
	buf := bytes.NewBuffer(newData)
	hash := make([]byte, 32)

	if _, err = buf.Read(hash); err != nil {
		return
	} else {
		e.ChainID.SetBytes(hash)
	}

	newData, err = e.FirstEntry.UnmarshalBinaryData(buf.Bytes())
	if err != nil {
		return
	}

	return
}

func (e *EChain) UnmarshalBinary(data []byte) (err error) {
	_, err = e.UnmarshalBinaryData(data)
	return
}
