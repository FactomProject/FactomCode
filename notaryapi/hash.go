package notaryapi

import (
	"bytes"
	"crypto/sha256"
)

type Hash struct {
	Bytes 		[]byte			`json:"bytes"`
}

func EmptyHash() (h *Hash) {
	h = new(Hash)
	h.Bytes = make([]byte, 32)
	return
}

func CreateHash(entities...BinaryMarshallable) (h *Hash, err error) {
	sha := sha256.New()
	
	for _, entity := range entities {
		data, err := entity.MarshalBinary()
		if err != nil { return nil, err }
		sha.Write(data)
	}
	
	h = new(Hash)
	h.Bytes = sha.Sum(nil)
	return
}

func (h *Hash) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	
	buf.Write([]byte{byte(len(h.Bytes))})
	buf.Write(h.Bytes)
	
	return buf.Bytes(), nil
}

func (h *Hash) MarshalledSize() uint64 {
	return uint64(len(h.Bytes)) + 1
}

func (h *Hash) UnmarshalBinary(data []byte) error {
	h.Bytes, data = make([]byte, data[0]), data[1:]
	copy(h.Bytes, data)
	
	return nil
}