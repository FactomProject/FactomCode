package main

import (
	"fmt"
)

var Entry struct {
	EntryType 		int64 = 1	// 1 is a plain entry
	StructuredData	[]byte		// The data (could be hashes) to record
	Signatures      [][]byte	// Optional signatures of the data
	TimeSamp        int64		// Unix Time
}

struct Encode ( entry Entry) []byte {
	

}


func main () {
	fmt.Print("Hello World")
}

