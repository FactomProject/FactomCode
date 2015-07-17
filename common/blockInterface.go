package common

import (
	"sync"
	"bytes"
)

type Chain struct {
	ChainID *Hash

	NextBlock       *Block
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

type NamedChain struct {
	Chain
	Name    [][]byte
}

type Block interface {
	GetHeader() Header
	GetBody()   Marshallable
	BuildHeader() error
}

type Marshallable interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) ([]byte, error)
	MarshalledSize() uint64
}

type Header interface {
	Marshallable
	BuildHeader(interface{}) error
}

func MarshalledSize(b Block) uint64 {
	return b.GetHeader().MarshalledSize() + b.GetBody().MarshalledSize()
}

func MarshalBinary(b Block) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := b.BuildHeader(); err != nil {
		return nil, err
	}
	if p, err := b.GetHeader().MarshalBinary(); err != nil {
		return nil, err
	} else {
		buf.Write(p)
	}

	if p, err := b.GetBody().MarshalBinary(); err != nil {
		return nil, err
	} else {
		buf.Write(p)
	}

	return buf.Bytes(), nil
}

func UnmarshalBinary(marshalled []byte, b Block) ([]byte, error) {
	rest, err:=b.GetHeader().UnmarshalBinary(marshalled)
	if err!=nil {
		return nil, err
	}
	rest, err=b.GetBody().UnmarshalBinary(rest)
	if err!=nil {
		return nil, err
	}
	return rest, nil
}