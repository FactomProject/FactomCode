package main

import (
	"github.com/firelizzard18/gocoding"
	"reflect"
)

type link struct {
	disp, url interface{}
}

func (l *link) Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		renderer.Printf(`<a href="%v">%v</a>`, l.url, l.disp)
	}
}
