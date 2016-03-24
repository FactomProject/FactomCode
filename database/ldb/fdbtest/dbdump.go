//
package main

import (
	"fmt"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/opt"
	"github.com/FactomProject/goleveldb/leveldb/util"
	//"bytes"
	//"encoding/binary"

	//	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/go-spew/spew"
)

type tst struct {
	key   uint32
	value string
}

// const dbpath = "/tmp/ldb9"
const dbpath = "/home/me/.btcd/data/factoid0/blocks_leveldb"

func main() {

	ro := &opt.ReadOptions{}
	//wo := &opt.WriteOptions{}
	opts := &opt.Options{}

	ldb, err := leveldb.OpenFile(dbpath, opts)
	fmt.Println("started db from ", dbpath)
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

		fmt.Printf("key: %v\n", common.EncodeBinary(&key))
		//		fmt.Println("  value:%v", iter.Value())

		buf := iter.Value()
		var buf2 []byte

		buf2 = buf

		fmt.Printf("value: ")
		//		fmt.Println(spew.Sdump(iter.Value()))
		spew.Dump(buf2)

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
