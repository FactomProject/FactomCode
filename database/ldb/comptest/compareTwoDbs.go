//
package main

import (
	"fmt"

	"bytes"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/goleveldb/leveldb"
	"github.com/FactomProject/goleveldb/leveldb/opt"
	"github.com/FactomProject/goleveldb/leveldb/util"
	//"encoding/binary"
)

type tst struct {
	key   uint32
	value string
}

const dbpath = "/tmp/ldb"
const dbpath2 = "/tmp/ldb9"

// This program is used to compare two databases - one from server and the other one from client
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

	ldb2, err := leveldb.OpenFile(dbpath2, opts)
	fmt.Println("started db2 from ", dbpath2)
	if err != nil {
		fmt.Printf("db2 open failed %v\n", err)
		return
	}

	elice := make([]*tst, 0, 10)

	iter := ldb.NewIterator(&util.Range{Start: nil, Limit: nil}, ro)

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		t := new(tst)

		value2, _ := ldb2.Get(key, ro)

		if value2 == nil {
			fmt.Printf("key:%v\n", common.EncodeBinary(&key))
		} else if bytes.Compare(value, value2) != 0 {
			fmt.Printf("Key with different values:%v\n", common.EncodeBinary(&key))
			fmt.Printf("value1:%v\n", common.EncodeBinary(&value))
			fmt.Printf("value2:%v\n", common.EncodeBinary(&value2))
		}

		elice = append(elice, t)

	}

	fmt.Printf("len(elice):%v", len(elice))
	fmt.Printf("completed\n")
	ldb.Close()
}
