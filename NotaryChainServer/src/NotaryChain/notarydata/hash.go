package notarydata

import (
	"hash"
)

type Hash struct {
	Bytes 		[]byte			`json:"bytes"`
}

func EmptyHash() (h *Hash) {
	h = new(Hash)
	h.Bytes = make([]byte, 32)
	return
}

func CreateHash(sha hash.Hash) (h *Hash) {
	h = new(Hash)
	h.Bytes = sha.Sum(nil)
	return
}

func (h *Hash) MarshalBinary() (data []byte, err error) {
	return
}

func (h *Hash) MarshalledSize() uint64 {
	return uint64(len(h.Bytes))
}

func (h *Hash) UnmarshalBinary(data []byte) error {
	return nil
}