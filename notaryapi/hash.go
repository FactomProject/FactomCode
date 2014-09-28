package notaryapi

import (
	"bytes"
	"crypto/sha256"
	"github.com/firelizzard18/gocoding"
	"reflect"

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

func (h *Hash) Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		hash := value.Interface().(*Hash)
		marshaller.MarshalObject(renderer, hash.Bytes)
	}
}

func (h *Hash) Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if value.IsNil() {
			value.Set(reflect.ValueOf(new(Hash)))
		}
		hash := value.Interface().(*Hash)
		unmarshaller.UnmarshalObject(scanner, &hash.Bytes)
	}
}
