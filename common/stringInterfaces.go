package common

import (
	"bytes"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
)

type JSONable interface {
	JSONByte() ([]byte, error)
	JSONString() (string, error)
	JSONBuffer(b *bytes.Buffer) error
}

type Spewable interface {
	Spew() string
}

type Printable interface {
	JSONable
	Spewable
}

func DecodeJSON(data []byte, v interface{}) error {
	err := json.Unmarshal(data, &v)
	return err
}

func EncodeJSON(data interface{}) ([]byte, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func EncodeJSONString(data interface{}) (string, error) {
	encoded, err := EncodeJSON(data)
	if err != nil {
		return "", err
	}
	return string(encoded), err
}

func DecodeJSONString(data string, v interface{}) error {
	return DecodeJSON([]byte(data), v)
}

func EncodeJSONToBuffer(data interface{}, b *bytes.Buffer) error {
	encoded, err := EncodeJSON(data)
	if err != nil {
		return err
	}
	_, err = b.Write(encoded)
	return err
}

func Spew(data interface{}) string {
	return spew.Sdump(data)
}
