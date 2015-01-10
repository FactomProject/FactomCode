package core

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type Entry struct {
	ChainID   *Hash
	ExtHashes []Hash
	Data      []byte
}

type Chain struct {
	ChainID *Hash
	Name    [][]byte
}

func (e *Entry) Hash() Hash {
}

func (c *Chain) GenerateID() (chainID Hash, err error) {
	byteSlice := make([]byte, 0, 32)
	for _, bytes := range c.Name {
		byteSlice = append(byteSlice, Sha(bytes).Bytes...)
	}
	c.ChainID = Sha(byteSlice)
	return c.ChainID
}
