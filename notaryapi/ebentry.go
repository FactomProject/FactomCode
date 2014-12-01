package notaryapi

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"time"
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
	binary.BigEndian.PutUint64(b, uint64(e.timeStamp)) 
	return b
	
}

func (e *EBEntry) SetTimeStamp(binaryTime []byte)  {
 	
 	e.timeStamp = int64(binary.BigEndian.Uint64(binaryTime))	

}

func (e *EBEntry) SetIntTimeStamp(ts int64)  {
 	
 	e.timeStamp = ts	

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