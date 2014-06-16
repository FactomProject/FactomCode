package notaryapi

import (
	"bytes"
	"errors"
	"io"
	"time"
	
	"encoding/base64"
	"encoding/binary"
	
	"github.com/firelizzard18/gocoding"
)

const (
	BadEntryType	= -1
	DataEntryType	=  0
)

type Entry interface {
	Type() int8
	Data() []byte
	TimeStamp() int64
	
	BinaryMarshallable
	TypeName() string
	RealTime() time.Time
	StampTime()
}

type SignedEntry interface {
	Entry
	Signatures() []Signature
	
	Sign(rand io.Reader, k PrivateKey) error
	Verify(k PublicKey, s int) bool
	Unsign(s int) bool
}

func EntryTypeName(entryType int8) string {
	switch entryType {
	case DataEntryType:
		return "Data"
		
	default:
		return "Unknown"
	}
}

func EntryTypeCode(entryType string) int8 {
	switch entryType {
	case "Data":
		return DataEntryType
		
	default:
		return BadEntryType
	}
}

func UnmarshalBinaryEntry(data []byte) (e Entry, err error) {
	switch int(data[0]) {
	case DataEntryType:
		e = new(DataEntry)
		
	default:
		return nil, errors.New("Bad entry type")
	}
	
	err = e.UnmarshalBinary(data)
	return
}

/* ----- ----- ----- ----- ----- */

type basicEntry struct {
	timeStamp int64
}

func makeBasicEntry() basicEntry {
	e := basicEntry{}
	e.StampTime()
	return e
}

func (e *basicEntry) Type() int8 {
	return BadEntryType
}

func (e *basicEntry) TypeName() string {
	return EntryTypeName(e.Type())
}

func (e *basicEntry) Data() []byte {
	return []byte{}
}

func (e *basicEntry) TimeStamp() int64 {
	return e.timeStamp
}

func (e *basicEntry) RealTime() time.Time {
	return time.Unix(e.TimeStamp(), 0)
}

func (e *basicEntry) StampTime() {
	e.timeStamp = time.Now().Unix()
}

func (e *basicEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	
	buf.Write([]byte{byte(e.Type())})
	
	data := e.Data()
	count := uint64(len(data))
	binary.Write(&buf, binary.BigEndian, count)
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, e.TimeStamp())
	
	return buf.Bytes(), nil
}

func (e *basicEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 1 // EntryType() int8
	size += 4 // len(Data()) uint64
	size += uint64(len(e.Data()))
	size += 4 // TimeStamp int64
	
	return size
}

func (e *basicEntry) UnmarshalBinary(data []byte) (err error) {
	data = data[1:] // don't care about type
	count, data := binary.BigEndian.Uint64(data[:4]), data[4:]
	data = data[count:] // let someone else parse the data
	
	e.timeStamp = int64(binary.BigEndian.Uint64(data[:4]))
	
	return nil
}

func (e *basicEntry) MarshallableFields() []gocoding.Field {
	return []gocoding.Field{
		gocoding.MakeField("type", e.TypeName, nil),
		gocoding.MakeField("timeStamp", e.RealTime, nil), 
	}
}
/* ----- ----- ----- ----- ----- */

type basicSignedEntry struct {
	basicEntry
	signatures []Signature
}

func makeBasicSignedEntry() basicSignedEntry {
	e := basicSignedEntry{makeBasicEntry(), []Signature{}}
	return e
}

func (e *basicSignedEntry) Signatures() []Signature {
	return e.signatures
}

func (e *basicSignedEntry) Sign(rand io.Reader, k PrivateKey) error {
	s, err := k.Sign(rand, e.Data())
	if err != nil { return err }
	
	e.signatures = append(e.signatures, s)
	return nil
}

func (e *basicSignedEntry) Verify(k PublicKey, s int) bool {
	if s < 0 || s >= len(e.signatures) {
		return false
	}
	
	return k.Verify(e.Data(), e.signatures[s])
}

func (e *basicSignedEntry) Unsign(s int) bool {
	if s < 0 || s >= len(e.signatures) {
		return false
	}
	
	e.signatures = append(e.signatures[:s], e.signatures[s+1:]...)
	return true
}

func (e *basicSignedEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	
	data, err := e.basicEntry.MarshalBinary()
	if err != nil { return nil, err }
	buf.Write(data)
	
	count := uint64(len(e.signatures))
	binary.Write(&buf, binary.BigEndian, count)
	for _, sig := range e.Signatures() {
		data, err = sig.MarshalBinary()
		if err != nil { return nil, err }
		buf.Write(data)
	}
	
	return buf.Bytes(), nil
}

func (e *basicSignedEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += e.basicEntry.MarshalledSize()
	size += 4 // len(Signatures()) uint64
	for _, sig := range e.Signatures() {
		size += sig.MarshalledSize()
	}
	
	return size
}

func (e *basicSignedEntry) UnmarshalBinary(data []byte) error {
	err := e.basicEntry.UnmarshalBinary(data)
	if err != nil { return err }
	data = data[e.basicEntry.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[:4]), data[4:]
	e.signatures = make([]Signature, count)
	for i := uint64(0); i < count; i++ {
		e.signatures[i], err = UnmarshalBinarySignature(data)
		if err != nil { return err }
		data = data[e.signatures[i].MarshalledSize():]
	}
	
	return nil
}

func (e *basicSignedEntry) MarshallableFields() []gocoding.Field {
	return append(e.basicEntry.MarshallableFields(),
		gocoding.MakeField("signatures", e.Signatures, nil))
}

/* ----- ----- ----- ----- ----- */

type DataEntry struct {
	basicSignedEntry
	data []byte
}

func MakeDataEntry() DataEntry {
	e := DataEntry{makeBasicSignedEntry(), []byte{}}
	return e
}

func (e *DataEntry) EntryType() int8 {
	return DataEntryType
}


func (e *DataEntry) Data() []byte {
	return e.data
}

func (e *DataEntry) DataBase64() string {
	return base64.StdEncoding.EncodeToString(e.Data())
}

func (e *DataEntry) UpdateData(data []byte) {
	e.data = data
}

func (e *DataEntry) UnmarshalBinary(data []byte) error {
	err := e.basicSignedEntry.UnmarshalBinary(data)
	if err != nil { return err }
	
	count, data := binary.BigEndian.Uint64(data[:4]), data[4:]
	e.data, data = data[:count], data[count:] // let someone else parse the data
	
	return nil
}

func (e *DataEntry) MarshallableFields() []gocoding.Field {
	return append(e.basicSignedEntry.MarshallableFields(),
		gocoding.MakeField("data", e.DataBase64, nil))
}
