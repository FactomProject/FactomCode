package notaryapi

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"time"
)


type FBEntry struct {
	timeStamp int64
	MerkleRoot *Hash	// the same MR in EBlockHeader
	ChainID *Hash 
	
	// not marshalllized
	hash *Hash
	eblock *Block
	status int8 //for future use??
}

//func NewFBEntry(h *Hash, id *Hash) *FBEntry {
func NewFBEntry(eb *Block, h *Hash) *FBEntry {
	e := &FBEntry{}
	e.StampTime()
	e.hash = h
	
	e.eblock = eb
	e.ChainID = eb.Chain.ChainID
	e.MerkleRoot = eb.Header.MerkleRoot
	
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
	binary.BigEndian.PutUint64(b, uint64(e.timeStamp)) 
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

func (e *FBEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	data, _ := e.ChainID.MarshalBinary()	
	buf.Write(data)
	
	data, _ = e.MerkleRoot.MarshalBinary()
	buf.Write(data)
	
	return buf.Bytes(), nil
}

func (e *FBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	size += e.ChainID.MarshalledSize()// Chain ID	
	size += e.MerkleRoot.MarshalledSize()
	return size
}

func (e *FBEntry) UnmarshalBinary(data []byte) (err error) {
	e.ChainID = new (Hash)
	e.ChainID.UnmarshalBinary(data[:33])
		
	e.MerkleRoot = new(Hash)
	e.MerkleRoot.UnmarshalBinary(data[33:])
	
	return nil
}


func (e *FBEntry) ShaHash() *Hash {
	byteArray, _ := e.MarshalBinary()
	return Sha(byteArray)
}

