package common

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func mix(v []uint64) {
    for i:= 0 ; i < 100; i++ {
        v1 := rand.Int()%len(v)
        v2 := rand.Int()%len(v)
        t := v[v1]
        v[v1]=v[v2]
        v[v2]=t
    }
}

func TestVarInt(test *testing.T) {
	fmt.Printf("---\nTestVarInt\n---\n")

	for i := 0; i < 1000; i++ {
		var out, out2 bytes.Buffer

		v := make([]uint64, 10)

		for j := 0; j < len(v); j++ {
			sw := rand.Int63() % 5
			switch sw {
			case 0:
				v[j] = uint64(rand.Int63() & 0xFF)
			case 1:
				v[j] = uint64(rand.Int63() & 0xFFFF)
			case 2:
				v[j] = uint64(rand.Int63() & 0xFFFFFFFF)
			case 3:
				v[j] = uint64(rand.Int63()) // Test lowerbit
			case 4:
				v[j] = uint64(rand.Int63() << 1) // Test signed bit
			}
		}

		mix(v[:])

		for j := 0; j < len(v); j++ {
			n, err := WriteVarInt(&out2, v[j])
			if err != nil {
				fmt.Println(n, err)
				test.Fail()
				return
			}
			//            fmt.Printf("%x ",v[j])
		}
		
		for j := 0; j < len(v); j++ {
			err := EncodeVarInt(&out, v[j])
			if err != nil {
				fmt.Println(err)
				test.Fail()
				return
			}
			//            fmt.Printf("%x ",v[j])
		}
		//        fmt.Println( "Length: ",out.Len())

		data := out.Bytes()
		
		//        PrtData(data)
		//        fmt.Println()

		var dv uint64
		for j := 0; j < len(v); j++ {
			dv, data = DecodeVarInt(data)
			if dv != v[j] {
				fmt.Printf("Values don't match: %x %x (%d)\n", dv, v[j], j)
				test.Fail()
				return
			}
		}
		
		//        PrtData(data)
		//        fmt.Println()

		for j := 0; j < len(v); j++ {
			dv = ReadVarInt(&out2)
			if dv != v[j] {
				fmt.Printf("Values don't match: %x %x (%d)\n", dv, v[j], j)
				test.Fail()
				return
			}
		}

	}
}
