package notaryapi

import (
	"bytes"
	"encoding/binary"
	"time"

)

type Entry struct {
	ChainID Hash	
	ExtHashes []Hash
	Data []byte
	
	timeStamp int64
}


type EntryInfo struct {

    EntryHash *Hash
    EBHash *Hash
    EBBlockNum uint64
    //EBOffset uint64
    
}

type EntryInfoBranch struct {
	EntryHash *Hash
	
	EntryInfo *EntryInfo
	EBInfo *EBInfo
	FBBatch *FBBatch
    
}

func (e *Entry) StampTime() {
	e.timeStamp = time.Now().Unix()
}
func (e *Entry) TimeStamp() int64 {
	return e.timeStamp
}

func (e *Entry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	data,_ := e.ChainID.MarshalBinary()
	buf.Write(data)
	
	count := uint8(len(e.ExtHashes))
	binary.Write(&buf, binary.BigEndian, count)
	for _, hash := range e.ExtHashes {
		data,_ = hash.MarshalBinary()
		buf.Write(data)		
	}	
	
	buf.Write(e.Data)

	return buf.Bytes(), nil
}

func (e *Entry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += e.ChainID.MarshalledSize() 	//33
	size += 1 							// 1 length of ExtHashes
	for _, hash := range e.ExtHashes { size += hash.MarshalledSize() }	
	size += uint64(len(e.Data))
	
	return size
}

func (e *Entry) UnmarshalBinary(data []byte) (err error) {
	e.ChainID.UnmarshalBinary(data[:33])
	data = data[33:]

	count,	data := data[0], data[1:]
	e.ExtHashes = make([]Hash, count)	
	for i := uint8(0); i < count; i++ {
		err := e.ExtHashes[i].UnmarshalBinary(data[:e.ExtHashes[i].MarshalledSize()])
		if err != nil { return err }
		data = data[e.ExtHashes[i].MarshalledSize():]
	}	
	
	e.Data = data
	
	return nil
}


func (e *EntryInfo) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	data, _ = e.EntryHash.MarshalBinary()
	buf.Write(data)
	
	data, _ = e.EBHash.MarshalBinary()
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, e.EBBlockNum)
	
	return buf.Bytes(), nil
}

func (e *EntryInfo) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 33	//e.EntryHash
	size += 33  //e.EBHash
	size += 8 	//e.EBBlockNum
	
	return size
}

func (e *EntryInfo) UnmarshalBinary(data []byte) (err error) {
	e.EntryHash = new(Hash)
	e.EntryHash.UnmarshalBinary(data[:33])

	data = data[33:]
	e.EBHash = new(Hash)
	e.EBHash.UnmarshalBinary(data[:33])	
	
	data = data[33:]
	e.EBBlockNum = binary.BigEndian.Uint64(data[0:8])
	
	return nil
}
