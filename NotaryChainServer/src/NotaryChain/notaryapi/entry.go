package notaryapi

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"time"
	
	"encoding/base64"
	"encoding/binary"
	
	"github.com/firelizzard18/gocoding"
)

type EntryData interface {
	Type() uint32
	Version() uint32
	Data() []byte
	DataEncoding(gocoding.Marshaller, reflect.Type) gocoding.Encoder
	UnmarshalBinary([]byte) error
}

const (
	EmptyDataType = 0
	PlainDataType = 1
)

func newEntryDataOfType(dataType uint32, version uint32) EntryData {
	switch {
	case dataType == PlainDataType && version == 0:
		return new(PlainData)
	
	default:
		return nil
	}
}

/* ----- ----- ----- ----- ----- */

type Entry struct {
	EntryData
	unixTime int64
	signatures []Signature
	
	BinaryMarshallable
}

func NewEntry(data EntryData) *Entry {
	e := &Entry{EntryData: data}
	e.StampTime()
	return e
}

func (e *Entry) TimeStamp() int64 {
	return e.unixTime
}

func (e *Entry) StampTime() {
	e.unixTime = time.Now().Unix()
}

func (e *PlainData) DataBase64() string {
	return base64.StdEncoding.EncodeToString(e.Data())
}

func (e *Entry) Signatures() []Signature {
	return e.signatures
}

func (e *Entry) IsSigned() bool {
	if e.signatures == nil {
		return false
	}
	
	return len(e.signatures) > 0
}

func (e *Entry) Sign(rand io.Reader, k PrivateKey) error {
	s, err := k.Sign(rand, e.Data())
	if err != nil { return err }
	
	e.signatures = append(e.signatures, s)
	return nil
}

func (e *Entry) Verify(k PublicKey, s int) bool {
	if e.signatures == nil || s < 0 || s >= len(e.signatures) {
		return false
	}
	
	return k.Verify(e.Data(), e.signatures[s])
}

func (e *Entry) Unsign(s int) bool {
	if s < 0 || s >= len(e.signatures) {
		return false
	}
	
	e.signatures = append(e.signatures[:s], e.signatures[s+1:]...)
	
	if len(e.signatures) == 0 {
		e.signatures = nil
	}
	
	return true
}

func (e *Entry) Encoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	dataEncoder := e.DataEncoding(marshaller, theType)
	
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*Entry)
		
		renderer.StartStruct()
		
		renderer.StartElement(`Type`)
		marshaller.MarshalObject(e.Type())
		renderer.StopElement(`Type`)
		
		renderer.StartElement(`Version`)
		marshaller.MarshalObject(e.Version())
		renderer.StopElement(`Version`)
		
		renderer.StartElement(`TimeStamp`)
		marshaller.MarshalObject(e.TimeStamp())
		renderer.StopElement(`TimeStamp`)
		
		dataEncoder(scratch, renderer, reflect.ValueOf(e.EntryData))
		
		renderer.StartElement(`Signatures`)
		marshaller.MarshalObject(e.Signatures())
		renderer.StopElement(`Signatures`)
		
		renderer.StopStruct()
	}
}

func (e *Entry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, e.Type())
	binary.Write(&buf, binary.BigEndian, e.Version())
	binary.Write(&buf, binary.BigEndian, e.TimeStamp())
	
	data := e.Data()
	binary.Write(&buf, binary.BigEndian, uint64(len(data)))
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

func (e *Entry) MarshalledSize() uint64 {
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

func (e *Entry) UnmarshalBinary(data []byte) (err error) {
	dataType,	data := binary.BigEndian.Uint32(data[:4]), data[4:]
	version,	data := binary.BigEndian.Uint32(data[:4]), data[4:]
	timeStamp,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	dataCount,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	entryData,	data := data[:dataCount], data[dataCount:]
	sigCount,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	
	e.EntryData = newEntryDataOfType(dataType, version)
	if e.EntryData == nil { return errors.New("Bad entry data type") }
	
	e.unixTime = int64(timeStamp)
	
	err = e.EntryData.UnmarshalBinary(entryData)
	if err != nil { return }
	
	e.signatures = make([]Signature, sigCount)
	for i := uint64(0); i < sigCount; i++ {
		e.signatures[i], err = UnmarshalBinarySignature(data)
		if err != nil { return err }
		data = data[e.signatures[i].MarshalledSize():]
	}
	
	return nil
}

/* ----- ----- ----- ----- ----- */

type PlainData struct {
	data []byte
}

func NewPlainData(data []byte) *PlainData {
	return &PlainData{data}
}

func NewDataEntry(data []byte) *Entry {
	return NewEntry(NewPlainData(data))
}

func (e *PlainData) Type() uint32 {
	return PlainDataType
}

func (e *PlainData) Version() uint32 {
	return 0
}

func (e *PlainData) Data() []byte {
	return e.data
}

func (e *PlainData) DataEncoding(marshaller gocoding.Marshaller, theType reflect.Type) gocoding.Encoder {
	return func(scratch [64]byte, renderer gocoding.Renderer, value reflect.Value) {
		e := value.Interface().(*PlainData)
		
		renderer.StartElement(`Data`)
		marshaller.MarshalObject(e.Data())
		renderer.StopElement(`Data`)
	}
}

func (e *PlainData) UnmarshalBinary(data []byte) (err error) {
	e.data = data
	return nil
}
