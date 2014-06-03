package data

import (
	"hash"
)

type Hash struct {
	bytes []byte
}

func CreateHash(w hash.Hash) (h *Hash) {
	h = new(Hash)
	h.bytes = w.Sum(nil)
	return h
}

func (h *Hash) writeToHash(w hash.Hash) (err error) {
	w.Write(h.bytes)
	return nil
}