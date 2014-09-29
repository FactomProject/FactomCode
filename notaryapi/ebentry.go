package notaryapi

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"time"

	"github.com/firelizzard18/gocoding"

	//"github.com/conformal/btcwire"
)


type EBEntry struct {
	timeStamp int64
	hash *Hash
	
	ChainID *[]byte // not marshalllized
	status int8 //for future use??
}

func NewEBEntry(h *Hash, id *[]byte) *EBEntry {
	e := &EBEntry{}
	e.StampTime()
	e.hash = h
	e.ChainID = id
	return e
}

func (e *EBEntry) Hash() *Hash {
	return e.hash
}

func (e *EBEntry) SetHash( binaryHash []byte)  {
	h := new(Hash)
	h.Bytes = binaryHash
	e.hash = h
}

func (e *EBEntry) TimeStamp() int64 {
	return e.timeStamp
}


func (e *EBEntry) GetBinaryTimeStamp() (binaryTimeStamp []byte)  {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(e.timeStamp)) //??
	return b
	
}

func (e *EBEntry) SetTimeStamp(binaryTime []byte)  {
 	
 	e.timeStamp = int64(binary.BigEndian.Uint64(binaryTime))	

}

func (e *EBEntry) RealTime() time.Time {
	return time.Unix(e.timeStamp, 0)
}

func (e *EBEntry) StampTime() {
	e.timeStamp = time.Now().Unix()
}

func (e *EBEntry) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`TimeStamp`: reflect.ValueOf(e.TimeStamp()),
		`Hash`: reflect.ValueOf(e.Hash()),
	}
	return fields
}

func (e *EBEntry) Decoding(unmarshaller gocoding.Unmarshaller, theType reflect.Type) gocoding.Decoder {
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
			case `type`:
				unmarshaller.UnmarshalObject(scanner, &typecode)
				
			case `timestamp`:
				unmarshaller.UnmarshalObject(scanner, &e.timeStamp)
				
			case `hash`:
				unmarshaller.UnmarshalObject(scanner, &e.hash)
				
			default:
			}
		}
	}
}

func (e *EBEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	
	binary.Write(&buf, binary.BigEndian, e.TimeStamp())
	
	data, _ := e.Hash().MarshalBinary()
	buf.Write(data)
	

	
	return buf.Bytes(), nil
}

func (e *EBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 // TimeStamp() int64	
	size += e.Hash().MarshalledSize()

	
	return size
}

func (e *EBEntry) UnmarshalBinary(data []byte) (err error) {

	
	timeStamp,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	e.timeStamp = int64(timeStamp)
		
	e.hash = new(Hash)
	e.hash.UnmarshalBinary(data)


	
	return nil
}


func (e *EBEntry) ShaHash() *Hash {
	byteArray, _ := e.MarshalBinary()
	return Sha(byteArray)
}

/*
// Sha generates the ShaHash name for the EBEntry.
func (e *EBEntry) Sha() *btcwire.ShaHash {
	//buf := bytes.NewBuffer(make([]byte, 0, msg.SerializeSize()))
	byteArray, _ := e.MarshalBinary()
	var sha btcwire.ShaHash
	_ = sha.SetBytes(btcwire.DoubleSha256(byteArray))

	return &sha
}
*/