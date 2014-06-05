package data

import (
	"hash"
	"encoding/binary"
	"crypto/sha256"
)

const (
	EmptyEntryType	= -1
	PlainEntryType	=  0
)

type Entry struct {
	EntryType		int8			`json:"entryType"`
}

type PlainEntry struct {
	Entry
	StructuredData	[]byte			`json:"structuredData"`				// The data (could be hashes) to record
	Signatures		[]*Signature	`json:"signatures" db:"-"`		// Optional signatures of the data
	TimeStamp		int64			`json:"timeStamp"`					// Unix Time
}

func (e *Entry) Hash() (hash *Hash, err error) {
	h := sha256.New()
	e.writeToHash(h)
	return CreateHash(h), nil
}

func (e *Entry) writeToHash(h hash.Hash) (err error) {
	return nil
}

func (e *PlainEntry) writeToHash(h hash.Hash) (err error) {
	if _, err = h.Write(e.StructuredData); err != nil {
		return err
	}
	
	for _,s := range e.Signatures {
		if err = s.writeToHash(h); err != nil {
			return err
		}
	}
	
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(e.TimeStamp))
	
	_, err = h.Write(buf)
	return err
}