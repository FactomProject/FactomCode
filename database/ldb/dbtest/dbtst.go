//
package main

import (
	"fmt"

	"github.com/conformal/goleveldb/leveldb"
	"github.com/conformal/goleveldb/leveldb/opt"
	"github.com/conformal/goleveldb/leveldb/util"
	//"bytes"
	//"encoding/binary"
)

type tst struct {
	key   uint32
	value string
}

var dataset = []tst{
	//var dataset = []struct { key int, value string } {
	{1, "one"},
	{2, "two"},
	{3, "three"},
	{4, "four"},
	{5, "five"},
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
/*
	batch := new(leveldb.Batch)
	for _, datum := range dataset {
		key := fmt.Sprintf("%v", datum.key)
		batch.Put([]byte(key), []byte(datum.value))
	}
	err = ldb.Write(batch, wo)



	for _, datum := range dataset {
		key := fmt.Sprintf("%v", datum.key)
		data, err := ldb.Get([]byte(key), ro)

		if err != nil {
			fmt.Printf("db read failed %v\n", err)
		}

		if string(data) != datum.value {
			fmt.Printf("mismatched data from db key %v val %v db %v", key, datum.value, data)
		}
	}
	
	*/
	elice := make([]*tst, 0, 10) 	
	
	//var fromkey [] byte = []byte{byte(2)} 		  	// Table Name (4 bytes)	
	//var tokey [] byte = []byte{byte(51)} 		  	// Table Name (4 bytes)		
	iter := ldb.NewIterator(&util.Range{Start: nil, Limit: nil}, ro)
	
	for iter.Next() {			
			key := iter.Key()
			fmt.Println("key:%v", key)	
			fmt.Println("  len(key):%v", len(key))	
			fmt.Println("  value:%v", iter.Value())
			fmt.Println("  len(val):%v", len(iter.Value()))
			t := new (tst)
			
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
