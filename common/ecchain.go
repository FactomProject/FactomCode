// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"sync"
)

type ECChain struct {
	ChainID         *Hash
	Name            [][]byte
	NextBlock       *ECBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

func NewECChain() *ECChain {
	c := new(ECChain)
	c.ChainID = NewHash()
	c.ChainID.SetBytes(EC_CHAINID)
	c.Name = make([][]byte, 0)
	return c
}

func (c *ECChain) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	if p, err := c.ChainID.MarshalBinary(); err != nil {
		return buf.Bytes(), err
	} else {
		buf.Write(p)
	}

	binary.Write(buf, binary.BigEndian, uint64(len(c.Name)))

	for _, v := range c.Name {
		binary.Write(buf, binary.BigEndian, uint64(len(v)))
		buf.Write(v)
	}

	return buf.Bytes(), nil
}

func (c *ECChain) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	hash := make([]byte, 32)
	
	if _, err := buf.Read(hash); err != nil {
		return err
	} else if err := c.ChainID.SetBytes(hash); err != nil {
		return err
	}
	
	count := uint64(0)
	if err := binary.Read(buf, binary.BigEndian, count); err != nil {
		return err
	}
	c.Name = make([][]byte, count)

	for _, name := range c.Name {
		var l uint64
		if err := binary.Read(buf, binary.BigEndian, &l); err != nil {
			return err
		}
		name = make([]byte, l)
		if _, err := buf.Read(name); err != nil {
			return err
		}
	}

	return nil
}
