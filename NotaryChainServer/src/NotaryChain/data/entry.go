package data

import (
	"hash"
	"encoding/binary"
)

const (
	EmptyEntryType	= -1
	HashEntryType	=  0
	PlainEntryType	=  1
)

type Entry struct {
	EntryType		int8
}

type HashEntry struct {
	Entry
	Hash			Hash		// The hash data
}

type PlainEntry struct {
	Entry
	StructuredData	[]byte		// The data (could be hashes) to record
	Signatures      []Signature	// Optional signatures of the data
	TimeSamp        int64		// Unix Time
}

func (e *Entry) writeToHash(h hash.Hash) (err error) {
	return nil
}

func (e *HashEntry) writeToHash(h hash.Hash) (err error) {
	return e.Hash.writeToHash(h)
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
	
	var buf []byte
	binary.BigEndian.PutUint64(buf, uint64(e.TimeSamp))
	
	_, err = h.Write(buf)
	return err
}