package notaryapi

import (
	"encoding/hex"
	"reflect"
	//"strings"

	"github.com/FactomProject/gocoding"
	"github.com/FactomProject/gocoding/json"
)

func EncodeBinary(bytes *[]byte) string {
	return hex.EncodeToString(*bytes)
}

func DecodeBinary(bytes *string) ([]byte, error) {
	return hex.DecodeString(*bytes)
}

func NewUnmarshaller(decoding gocoding.Decoding) gocoding.Unmarshaller {
	return gocoding.NewUnmarshaller(NewDecoding(decoding))
}

func NewJSONUnmarshaller() gocoding.Unmarshaller {
	return NewUnmarshaller(json.Decoding)
}

func UnmarshalJSON(reader gocoding.SliceableRuneReader, obj interface{}) error {
	return NewJSONUnmarshaller().Unmarshal(json.Scan(reader), obj)
}

//var signatureType = reflect.TypeOf(new(Signature)).Elem()

func NewDecoding(decoding gocoding.Decoding) gocoding.Decoding {
	return func(m gocoding.Unmarshaller, t reflect.Type) gocoding.Decoder {
		return decoding(m, t)
	}
}
