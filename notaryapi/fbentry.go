package notaryapi

import (
	"bytes"
	"encoding/binary"
	//"encoding/hex"
	//"errors"
	"github.com/firelizzard18/gocoding"
	//"io"
	"reflect"
	//"regexp"
	"strings"
	"time"
)


type FBEntry struct {
	timeStamp int64
	hash *Hash
	
	
	ChainID *[]byte // not marshalllized
	status int8 //for future use??
}

func NewFBEntry(h *Hash, id *[]byte) *FBEntry {
	e := &FBEntry{}
	e.StampTime()
	e.hash = h
	e.ChainID = id
	return e
}

func (e *FBEntry) Hash() *Hash {
	return e.hash
}

func (e *FBEntry) SetHash( binaryHash []byte)  {
	h := new(Hash)
	h.Bytes = binaryHash
	e.hash = h
}

func (e *FBEntry) TimeStamp() int64 {
	return e.timeStamp
}


func (e *FBEntry) GetBinaryTimeStamp() (binaryTimeStamp []byte)  {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(e.timeStamp)) //??
	return b
	
}

func (e *FBEntry) SetTimeStamp(binaryTime []byte)  {
 	
 	e.timeStamp = int64(binary.BigEndian.Uint64(binaryTime))	

}

func (e *FBEntry) RealTime() time.Time {
	return time.Unix(e.timeStamp, 0)
}

func (e *FBEntry) StampTime() {
	e.timeStamp = time.Now().Unix()
}

func (e *FBEntry) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`TimeStamp`: reflect.ValueOf(e.TimeStamp()),
		`Hash`: reflect.ValueOf(e.Hash()),
	}
	return fields
}

func (e *FBEntry) Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
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
			value.Set(reflect.ValueOf(new(EBEntry)))
		}
		
		e := value.Interface().(*EBEntry)
		typecode := EmptyDataType
		
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
			case `chainid`:
				unmarshaller.UnmarshalObject(scanner, &typecode)//??
				
			case `hash`:
				unmarshaller.UnmarshalObject(scanner, &e.hash)
				
			default:
			}
		}
	}
}

func (e *FBEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	
	buf.Write(*e.ChainID)
	
	data, _ := e.Hash().MarshalBinary()
	buf.Write(data)
	

	
	return buf.Bytes(), nil
}

func (e *FBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 32 // Chain ID	
	size += e.Hash().MarshalledSize()

	
	return size
}

func (e *FBEntry) UnmarshalBinary(data []byte) (err error) {

	
	chainID := data[:32]
	e.ChainID = &chainID
		
	e.hash = new(Hash)
	e.hash.UnmarshalBinary(data[32:])
	
	return nil
}

