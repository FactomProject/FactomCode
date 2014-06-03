package data

import (
	"hash"
)

type Hash struct {
	bytes []byte
}

func (h *Hash) writeToHash(w hash.Hash) (err error) {
	w.Write(h.bytes)
	return nil
}