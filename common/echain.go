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

func (e *EChain) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)

	if _, err := buf.Read(hash); err != nil {
		return err
	} else {
		e.ChainID.SetBytes(hash)
	}
	
	if err := e.FirstEntry.UnmarshalBinary(buf.Bytes()); err !=  nil {
		return err
	}
	
	return nil
}