// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package process

import (
	"fmt"
	"time"
	fct "github.com/FactomProject/factoid"
	"math/rand"
	"testing"
)

var _ = fmt.Printf
var _ = rand.New

var now = time.Now().Unix()
var hour = int64(60*60)

func Test_Replay(test *testing.T) {
	
	type mh struct { 
		hash []byte 
		time int64 
	}
	
	var h [5000] *mh
	
	for i := 0; i<5000; i++ {
		h[i] = new(mh)
		h[i].hash = fct.Sha([]byte(fmt.Sprintf("h%d",i))).Bytes()
		h[i].time = now + (rand.Int63()%24*hour)-12*hour
		
		if !IsTSValid_(h[i].hash, h[i].time,now) { 
			fmt.Println("Failed Test ",i,"first")
			test.Fail() 
			return
		}
		if IsTSValid_(h[i].hash, h[i].time,now) { 
			fmt.Println("Failed Test ",i,"second")
			test.Fail()
			return
		}
		now += rand.Int63()%hour
		for j:=0; j<i; j++ {
			if IsTSValid_(h[i].hash, h[i].time,hour) {
				fmt.Println("Failed Test ",i,j,"repeat")
				test.Fail()
				return
			}
		}
	}
			
}