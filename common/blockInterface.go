package common

import (
	"bytes"
	"encoding"
	"sync"
)

type Chain struct {
	ChainID *Hash

	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

type NamedChain struct {
	Chain
	Name [][]byte
}

type Block interface {
	GetHeader() Header
	GetBody() Body
	BuildHeader() error
}

type Marshallable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	UnmarshalBinaryReturnRest([]byte) ([]byte, error)
	MarshalledSize() uint64
}

type Header interface {
	Marshallable
	//We are passing Body to be able to figure out length of arrays
	BuildHeader(interface{}) error
}

type Body interface {
	MarshalBinary() ([]byte, error)
	//We are passing Header to be able to figure out length of arrays
	UnmarshalBinaryReturnRest([]byte, interface{}) ([]byte, error)
	MarshalledSize() uint64
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

func UnmarshalBinary(b Block, marshalled []byte) ([]byte, error) {
	rest, err := b.GetHeader().UnmarshalBinaryReturnRest(marshalled)
	if err != nil {
		return nil, err
	}
	rest, err = b.GetBody().UnmarshalBinaryReturnRest(rest, b.GetHeader())
	if err != nil {
		return nil, err
	}
	return rest, nil
}
