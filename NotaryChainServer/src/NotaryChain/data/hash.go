package data

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

func (h *Hash) writeToHash(sha hash.Hash) (err error) {
	sha.Write(h.Bytes)
	return
}