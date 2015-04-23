package notaryapi

import (
	"fmt"
	"testing"
)

func TestMarshall(t *testing.T) {
	dbinfo := new(DBInfo)
	
	dbinfo.DBHash = NewHash()
	dbinfo.DBHash.Bytes = []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	dbinfo.BTCTxHash = NewHash()
	dbinfo.BTCTxHash.Bytes = []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	dbinfo.BTCTxOffset = 10
	dbinfo.BTCBlockHeight = 20
	dbinfo.BTCBlockHash = NewHash()
	dbinfo.BTCBlockHash.Bytes = []byte("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	dbinfo.DBMerkleRoot = NewHash()
	dbinfo.DBMerkleRoot.Bytes = []byte("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")

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