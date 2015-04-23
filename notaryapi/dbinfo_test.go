package notaryapi

import (
	"bytes"
	"fmt"
	"testing"
)

type tDBInfo struct {
	DBHash       *Hash
	BTCTxHash    *Hash
	BTCBlockHash *Hash
	DBMerkleRoot *Hash
}

func (d *tDBInfo) MarshalBinary() ([]byte, error) {
	var (
		buf  bytes.Buffer
		data []byte
		err  error
	)
	
	data, err = d.DBHash.MarshalBinary()
	if err != nil {
		return buf.Bytes(), err
	}
	buf.Write(data)
	
	data, err = d.BTCTxHash.MarshalBinary()
	if err != nil {
		return buf.Bytes(), err
	}
	buf.Write(data)
	
	data, err = d.BTCBlockHash.MarshalBinary()
	if err != nil {
		return buf.Bytes(), err
	}
	buf.Write(data)
	
	data, err = d.DBMerkleRoot.MarshalBinary()
	if err != nil {
		return buf.Bytes(), err
	}
	buf.Write(data)
	
	return buf.Bytes(), err
}

func (d *tDBInfo) UnmarshalBinary(data []byte) (err error) {
	d.DBHash.UnmarshalBinary(data[:33])
	return
}

func TestMarshall(t *testing.T) {
	dbinfo := new(DBInfo)
	
	dbinfo.DBHash = NewHash()
	dbinfo.DBHash.Bytes = []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	dbinfo.BTCTxHash = NewHash()
	dbinfo.BTCTxHash.Bytes = []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	dbinfo.BTCTxOffset = 10
	dbinfo.BTCBlockHeight = 20
	dbinfo.BTCBlockHash = NewHash()
	dbinfo.BTCBlockHash.Bytes = []byte("cccccccccccccccccccccccccccccccc")
	dbinfo.DBMerkleRoot = NewHash()
	dbinfo.DBMerkleRoot.Bytes = []byte("dddddddddddddddddddddddddddddddd")

	fmt.Println(dbinfo)

	p, err := dbinfo.MarshalBinary()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(p))
	
	err = dbinfo.UnmarshalBinary(p)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(dbinfo)
	
	p, err = dbinfo.MarshalBinary()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(p))
}