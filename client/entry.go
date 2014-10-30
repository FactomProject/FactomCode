package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/firelizzard18/gocoding"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"
	"github.com/FactomProject/FactomCode/notaryapi"
	
)

/*
type BEntry interface {
	ShaHash() *Hash
}
 */

type EntryData interface {
	Type() EntryDataType
	TypeName() string
	Version() EntryDataVersion
	Data() []byte
	
	EncodeToString() string
	DecodeFromString(string) error
	
	EncodableFields() map[string]reflect.Value
	FieldDecoding(gocoding.Unmarshaller, string) gocoding.Decoder
	UnmarshalBinary([]byte) error
}


type EntryDataType uint32
type EntryDataVersion uint32

const NoDataVersion EntryDataVersion = 0

const (
	EmptyDataType EntryDataType = iota 
	PlainDataType 
	UTF8DataType
)

func NewEntryOfType(dataType EntryDataType) *ClientEntry {
	switch dataType {
	case PlainDataType:
		return NewEntry(new(plainData))
		
	case UTF8DataType:
		return NewEntry(&_UTF8Data{new(plainData)})
	
	default:
		return nil
	}
}

func newEntryDataOfType(dataType EntryDataType, version EntryDataVersion) EntryData {
	switch {
	case dataType == PlainDataType && version == PlainDataVersion1:
		return new(plainData)
		
	case dataType == UTF8DataType && version == PlainDataVersion1:
		return new(_UTF8Data)
	
	default:
		return nil
	}
}


type ClientEntry struct {
	ChainID []byte
	ExtHashes *[]notaryapi.Hash	
	EntryData
	unixTime int64

	signatures []notaryapi.Signature	
	
	notaryapi.BinaryMarshallable
}


func NewEntry(data EntryData) *ClientEntry {
	e := &ClientEntry{EntryData: data}
	e.StampTime()
	return e
}

func (e *ClientEntry) TimeStamp() int64 {
	return e.unixTime
}

func (e *ClientEntry) Type() EntryDataType {
	if e.EntryData == nil {
		return EmptyDataType
	} else {
		return e.EntryData.Type()
	}
}

func (e *ClientEntry) Version() EntryDataVersion {
	if e.EntryData == nil {
		return NoDataVersion
	} else {
		return e.EntryData.Version()
	}
}

func (e *ClientEntry) RealTime() time.Time {
	return time.Unix(e.unixTime, 0)
}

func (e *ClientEntry) StampTime() {
	e.unixTime = time.Now().Unix()
}

func (e *ClientEntry) Signatures() []notaryapi.Signature {
	return e.signatures
}

func (e *ClientEntry) IsSigned() bool {
	if e.signatures == nil {
		return false
	}
	
	return len(e.signatures) > 0
}

func (e *ClientEntry) Sign(rand io.Reader, k notaryapi.PrivateKey) error {
	s, err := k.Sign(rand, e.Data())
	if err != nil { return err }
	
	e.signatures = append(e.signatures, s)
	return nil
}

func (e *ClientEntry) Verify(k notaryapi.PublicKey, s int) bool {
	if e.signatures == nil || s < 0 || s >= len(e.signatures) {
		return false
	}
	
	return k.Verify(e.Data(), e.signatures[s])
}

func (e *ClientEntry) Unsign(s int) bool {
	if s < 0 || s >= len(e.signatures) {
		return false
	}
	
	e.signatures = append(e.signatures[:s], e.signatures[s+1:]...)
	
	if len(e.signatures) == 0 {
		e.signatures = nil
	}
	
	return true
}

func (e *ClientEntry) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`Type`: reflect.ValueOf(e.Type()),
		`Version`: reflect.ValueOf(e.Version()),
		`TimeStamp`: reflect.ValueOf(e.TimeStamp()),
		`Signatures`: reflect.ValueOf(e.Signatures()),
	}
	
	for name, value := range e.EntryData.EncodableFields() {
		fields[name] = value
	}
	
	return fields
}

func (e *ClientEntry) Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
	return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
		if scanner.Peek() == gocoding.ScannedLiteralBegin {
			null := scanner.NextValue()
			if null.IsValid() && null.IsNil() {
				value.Set(reflect.Zero(theType))
				return
			}
		}
	
		if !gocoding.PeekCheck(scanner, gocoding.ScannedStructBegin, gocoding.ScannedMapBegin) { return }
		
		if value.IsNil() {
			value.Set(reflect.ValueOf(new(ClientEntry)))
		}
		
		e := value.Interface().(*ClientEntry)
		data := value.Elem().FieldByName(`EntryData`)
		typecode := EmptyDataType
		version := NoDataVersion
		settype := false
		setver := false
		
		for {
			// get the next code, check for the end
			code := scanner.Continue()
			if code.Matches(gocoding.ScannedStructEnd, gocoding.ScannedMapEnd) { break }
			
			// check for key begin
			if code != gocoding.ScannedKeyBegin {
				// this will generate an appropriate error message
				gocoding.PeekCheck(scanner, gocoding.ScannedKeyBegin, gocoding.ScannedStructEnd, gocoding.ScannedMapEnd)
				return
			}
			
			// get the key
			key := scanner.NextValue()
			if key.Kind() != reflect.String {
				scanner.Error(gocoding.ErrorPrintf("Decoding", "Invalid key type %s", key.Type().String()))
				return
			}
			keystr := key.String()
			
			scanner.Continue()
			switch strings.ToLower(keystr) {
			case `type`:
				settype = true
				unmarshaller.UnmarshalObject(scanner, &typecode)
				
				if e.EntryData != nil && e.Type() != typecode {
					scanner.Error(gocoding.ErrorPrintf("Decoding", "Expected type code %d, got %d", e.Type(), typecode))
					return
				}
				
			case `version`:
				setver = true
				unmarshaller.UnmarshalObject(scanner, &version)
				
				if e.EntryData != nil && e.Version() != version {
					scanner.Error(gocoding.ErrorPrintf("Decoding", "Expected version %d, got %d", e.Version(), version))
					return
				}
				
			case `timestamp`:
				unmarshaller.UnmarshalObject(scanner, &e.unixTime)
				
			case `signatures`:
				unmarshaller.UnmarshalObject(scanner, &e.signatures)
				
			default:
				if e.EntryData == nil && !settype && !setver {
					scanner.Error(gocoding.ErrorPrintf("Decoding", "Can't unmarshall field %s without entry data type & version", keystr))
					return
				}
				
				if e.EntryData == nil {
					e.EntryData = newEntryDataOfType(typecode, version)
					if e.EntryData == nil {
						scanner.Error(gocoding.ErrorPrintf("Decoding", "Bad type/version: %d/%d", typecode, version))
						return
					}
				}
				
				decoder := e.FieldDecoding(unmarshaller, keystr)
				if decoder != nil {
					decoder(scratch, scanner, data)
				} else {
					scanner.NextValue()
				}
			}
		}
	}
}

func (e *ClientEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, e.Type())
	binary.Write(&buf, binary.BigEndian, e.Version())
	binary.Write(&buf, binary.BigEndian, e.TimeStamp())
	
	datacount := uint64(len(e.Data()))
	binary.Write(&buf, binary.BigEndian, datacount)	
	
	data := e.Data()
	buf.Write(data)
	
	count := uint64(len(e.Signatures()))
	binary.Write(&buf, binary.BigEndian, count)
	for _, sig := range e.Signatures() {
		data, err := sig.MarshalBinary()
		if err != nil { return nil, err }
		buf.Write(data)
	}
	
	return buf.Bytes(), nil
}

func (e *ClientEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 4 // Type() uint32
	size += 4 // Version() uint32
	size += 8 // TimeStamp() int64
	size += 8 // len(Data()) uint64
	size += uint64(len(e.Data()))
	size += 8 // len(Signatures()) uint64
	for _, sig := range e.Signatures() { size += sig.MarshalledSize() }
	
	return size
}

func (e *ClientEntry) UnmarshalBinary(data []byte) (err error) {
	dataType,	data := binary.BigEndian.Uint32(data[:4]), data[4:]
	version,	data := binary.BigEndian.Uint32(data[:4]), data[4:]
	timeStamp,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	dataCount,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	entryData,	data := data[:dataCount], data[dataCount:]
	sigCount,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	

	e.EntryData = newEntryDataOfType(EntryDataType(dataType), EntryDataVersion(version))
	if e.EntryData == nil { return errors.New("Bad entry data type") }
	
	e.unixTime = int64(timeStamp)

	err = e.EntryData.UnmarshalBinary(entryData)
	if err != nil { return }
	
	e.signatures = make([]notaryapi.Signature, sigCount)
	for i := uint64(0); i < sigCount; i++ {
		e.signatures[i], err = notaryapi.UnmarshalBinarySignature(data)
		if err != nil { return err }
		data = data[e.signatures[i].MarshalledSize():]
	}
	
	return nil
}




var whitesp = regexp.MustCompile(`\s`)
var fieldDecoders = make(map[string]gocoding.Decoder)

const (
	_ EntryDataVersion = iota
	PlainDataVersion1
)

type plainData struct {
	data []byte
}

func NewPlainData(data []byte) EntryData {
	return &plainData{data}
}

func NewDataEntry(data []byte) *ClientEntry {
	return NewEntry(NewPlainData(data))
}

func (e *plainData) Type() EntryDataType {
	return PlainDataType
}

func (e *plainData) TypeName() string {
	return "Plain Data"
}

func (e *plainData) Version() EntryDataVersion {
	return PlainDataVersion1
}

func (e *plainData) Data() []byte {
	return e.data
}
	
func (e *plainData) EncodeToString() string {
	return hex.EncodeToString(e.Data())
}

func (e *plainData) DataHash() *notaryapi.Hash {
	if e.Data == nil{
		return notaryapi.EmptyHash()
	}
	sData := new (notaryapi.SimpleData)
	sData.Data = e.Data()
	hash, _ := notaryapi.CreateHash(sData)
	return hash
}

func (e *plainData) DecodeFromString(str string) error {
	str = whitesp.ReplaceAllString(str, "")
	if len(str) % 2 == 1 {
		str = "0" + str 
	}
	
	data, err := hex.DecodeString(str)
	if err != nil {
		return err
	}
	e.data = data
	return nil
}

func (e *plainData) EncodableFields() map[string]reflect.Value {
	return map[string]reflect.Value{`Data`: reflect.ValueOf(e.Data())}
}

func (e *plainData) FieldDecoding(unmarshaller gocoding.Unmarshaller, name string) gocoding.Decoder {
	decoder, ok := fieldDecoders[name]
	if ok {
		return decoder
	}
	
	switch name {
	case `Data`:
		decoder = func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
			if value.IsNil() {
				value.Set(reflect.ValueOf(new(plainData)))
			}
			
			e := value.Interface().(*plainData)
			unmarshaller.UnmarshalObject(scanner, &e.data)
		}
		
	default:
		return nil
	}
	
	fieldDecoders[name] = decoder
	return decoder
}

func (e *plainData) UnmarshalBinary(data []byte) (err error) {

	e.data = data

	return nil
}

/* ----- ----- ----- ----- ----- */

type _UTF8Data struct {
	*plainData
}

func NewUTF8Data(str string) EntryData {
	return &_UTF8Data{NewPlainData([]byte(str)).(*plainData)}
}

func NewUTF8Entry(str string) *ClientEntry {
	return NewEntry(NewUTF8Data(str))
}

func (e *_UTF8Data) Type() EntryDataType {
	return UTF8DataType
}

func (e *_UTF8Data) TypeName() string {
	return "UTF8 Data"
}
	
func (e *_UTF8Data) EncodeToString() string {
	return string(e.Data())
}

func (e *_UTF8Data) DecodeFromString(str string) error {
	e.plainData = NewPlainData([]byte(str)).(*plainData)
	return nil
}

func (e *_UTF8Data) FieldDecoding(unmarshaller gocoding.Unmarshaller, name string) gocoding.Decoder {
	decoder := (&plainData{}).FieldDecoding(unmarshaller, name)
	
	if decoder != nil {
		return func(scratch [64]byte, scanner gocoding.Scanner, value reflect.Value) {
//			if value.IsNil() {
//				value.Set(reflect.ValueOf(new(_UTF8Data)))
//			}
			e := value.Interface().(*_UTF8Data)
			value = reflect.ValueOf(&e.plainData)
			decoder(scratch, scanner, value.Elem())
		}
	}
	
	return decoder
}