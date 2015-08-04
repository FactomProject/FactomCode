package common_test

import (
    "fmt"
    "bytes"
	"math/rand"
    "testing"

	"github.com/FactomProject/FactomCode/common"
)

func TestVarInt(t *testing.T) {
	for i := 0; i < 1000; i++ {
		var out bytes.Buffer

		v := make([]uint64, 10)

		for j := 0; j < len(v); j++ {
			var m uint64           // 64 bit mask
			sw := rand.Int63() % 4 // Pick a random choice
			switch sw {
			case 0:
				m = 0xFF // Random byte
			case 1:
				m = 0xFFFF // Random 16 bit integer
			case 2:
				m = 0xFFFFFFFF // Random 32 bit integer
			case 3:
				m = 0xFFFFFFFFFFFFFFFF // Random 64 bit integer
			}
			n := uint64(rand.Int63() + (rand.Int63() << 32))
			v[j] = n & m
		}

		for j := 0; j < len(v); j++ { // Encode our entire array of numbers
			err := common.EncodeVarInt(&out, v[j])
			if err != nil {
				fmt.Println(err)
				t.Fail()
				return
			}
			//              fmt.Printf("%x ",v[j])
		}
		//          fmt.Println( "Length: ",out.Len())

		data := out.Bytes()

		//          PrtData(data)
		//          fmt.Println()
		sdata := data // Decode our entire array of numbers, and
		var dv uint64 // check we got them back correctly.
		for k := 0; k < 1000; k++ {
			data = sdata
			for j := 0; j < len(v); j++ {
				dv, data = common.DecodeVarInt(data)
				if dv != v[j] {
					fmt.Printf("Values don't match: decode:%x expected:%x (%d)\n", dv, v[j], j)
					t.Fail()
					return
				}
			}
		}
	}
}
    