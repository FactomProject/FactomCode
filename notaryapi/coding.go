package notaryapi

import (
	"github.com/firelizzard18/gocoding"
	"reflect"
	"strings"
)

var signatureType = reflect.TypeOf(new(Signature)).Elem()

func NewDecoding(decoding gocoding.Decoding) gocoding.Decoding {
	return func(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
		if !theType.ConvertibleTo(signatureType) {
			return decoding(unmarshaller, theType)
		}
		
		decoder := (&ECDSASignature{}).ElementDecoding(unmarshaller, theType)
		
		return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
			if scanner.Peek() == gocoding.ScannedLiteralBegin {
				null := scanner.NextValue()
				if null.IsValid() && null.IsNil() {
					value.Set(reflect.Zero(theType))
					return
				}
			}
		
			if !gocoding.PeekCheck(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
			
			// get the next code, should have at least a type key
			code := scanner.Continue()
			if code != gocoding.ScannedKeyBegin {
				gocoding.PeekCheck(scanner, gocoding.ScannedKeyBegin)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				scanner.Error(gocoding.ErrorPrint("Decoding", "Invalid key type %s", key.Type().String()))
				return
			}
			if str := strings.ToLower(key.String()); str != `type` {
				scanner.Error(gocoding.ErrorPrintf("Decoding", "Decoding signature: expected 'type', got '%s'", str))
				return
			}
			
			scanner.Continue()
			_type := scanner.NextValue()
			switch _type.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				 
			default:
				scanner.Error(gocoding.ErrorPrint("Decoding", "Invalid type code type %s", _type.Type().String()))
			}
			
			switch _type.Int() {
			case ECDSASignatureType:
				value.Set(reflect.ValueOf(&ECDSASignature{}))
				decoder(scratch, scanner, value)
			}
		}
	}
}