package notarydata

import (
	"encoding/binary"
	"bytes"
)

const (
	EmptyEntryType	= -1
	PlainEntryType	=  0
)

type Entry struct {
	EntryType		int8			`json:"entryType"`
}

func EntryTypeName(entryType int8) string {
	switch entryType {
	case EmptyEntryType:
		return "Empty"
	
	case PlainEntryType:
		return "Plain"
		
	default:
		return "Unknown"
	}
}

func EntryTypeCode(entryType string) int8 {
	switch entryType {
	case "Plain":
		return PlainEntryType
		
	default:
		return EmptyEntryType
	}
}

type PlainEntry struct {
	Entry
	StructuredData	[]byte			`json:"structuredData"`	// The data (could be hashes) to record
	Signatures		[]Signature		`json:"signatures"`	// Optional signatures of the data
	TimeStamp		int64			`json:"timeStamp"`	// Unix Time
}

func (e *PlainEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	buf.Write([]byte{byte(e.EntryType)})
	
	sdlen := uint64(len(e.StructuredData))
	binary.Write(&buf, binary.BigEndian, sdlen)
	buf.Write(e.StructuredData)
	
	count := uint64(len(e.Signatures))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i = i + 1 {
		data, err = e.Signatures[i].MarshalBinary()
		if err != nil { return }
		buf.Write(data)
	}
	
	binary.Write(&buf, binary.BigEndian, e.TimeStamp)
	
	return buf.Bytes(), err
}

func (e *PlainEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 1 // EntryType int8
	size += 4 // len(StructuredData) uint64
	size += uint64(len(e.StructuredData))
	size += 4 // len(Signatures) uint64
	size += 4 // TimeStamp int64
	
	for _, sig := range e.Signatures {
		size += sig.MarshalledSize()
	}
	
	return size
}

func (e *PlainEntry) UnmarshalBinary(data []byte) (err error) {
	e.EntryType, data = int8(data[0]), data[0:]
	
	sdlen, data := binary.BigEndian.Uint64(data[0:4]), data[4:]
	e.StructuredData = make([]byte, sdlen)
	copy(e.StructuredData, data)
	
	count, data := binary.BigEndian.Uint64(data[0:4]), data[4:]
	e.Signatures = make([]Signature, 0, count)
	for i := uint64(0); i < count; i = i + 1 {
		e.Signatures[0], err = UnmarshalBinarySignature(data)
		if err != nil { return }
		data = data[e.Signatures[0].MarshalledSize():]
	}
	
	e.TimeStamp = int64(binary.BigEndian.Uint64(data[0:4]))
	
	return nil
}
