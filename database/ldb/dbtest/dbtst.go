//
package main

import (
	"fmt"

	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/opt"
	"github.com/FactomProject/goleveldb/leveldb/util"
	//"bytes"
	//"encoding/binary"
)

type tst struct {
	key   uint32
	value string
}

func main() {

	ro := &opt.ReadOptions{}
	//wo := &opt.WriteOptions{}
	opts := &opt.Options{}

	ldb, err := leveldb.OpenFile("/tmp/ldb9", opts)
	fmt.Println("started db from /tmp/ldb9")
	if err != nil {
		fmt.Printf("db open failed %v\n", err)
		return
	}

	elice := make([]*tst, 0, 10)

	//var fromkey [] byte = []byte{byte(2)} 		  	// Table Name (4 bytes)
	//var tokey [] byte = []byte{byte(51)} 		  	// Table Name (4 bytes)
	iter := ldb.NewIterator(&util.Range{Start: nil, Limit: nil}, ro)

	for iter.Next() {
		key := iter.Key()
		fmt.Println("key:%v", notaryapi.EncodeBinary(&key))
		fmt.Println("  value:%v", iter.Value())
		t := new(tst)

		//t.key = binary.BigEndian.Uint32(key[:4])
		//buf := bytes.NewBuffer(key)
		//binary.Read(buf, binary.BigEndian, &t.key)

		//fmt.Println("t.key:%v", t.key)
		elice = append(elice, t)

	}

	fmt.Printf("len(elice):%v", len(elice))
	fmt.Printf("completed\n")
	ldb.Close()
}
