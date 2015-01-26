package factomapi

import (
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/gocoding"
	"github.com/FactomProject/gocoding/html"
	"github.com/FactomProject/gocoding/json"
	"io"
)

func SafeMarshal(writer io.Writer, obj interface{}) error {
	marshaller := json.NewMarshaller()
	renderer := json.Render(writer)
	renderer.SetRecoverHandler(safeRecover)
	return marshaller.Marshal(renderer, obj)
}

func SafeMarshalHTML(writer io.Writer, obj interface{}) error {
	renderer := html.Render(writer)
	renderer.SetRecoverHandler(safeRecover)

	marshallerHTML := html.NewMarshaller()
	return marshallerHTML.Marshal(renderer, obj)
}

func SafeUnmarshal(reader gocoding.SliceableRuneReader, obj interface{}) error {
	scanner := json.Scan(reader)
	scanner.SetRecoverHandler(EXPLODE)
	unmarshaller := notaryapi.NewJSONUnmarshaller()
	return unmarshaller.Unmarshal(scanner, obj)
}

func safeRecover(obj interface{}) error {
	if err, ok := obj.(error); ok {
		return err
	}

	return errors.New(fmt.Sprint(obj))
}

func EXPLODE(obj interface{}) error {
	panic(obj)
}
