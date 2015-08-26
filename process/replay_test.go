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


func Test_Replay(test *testing.T) {
	
	type mh struct { 
		hash []byte 
		time int64 
	}
	
	var h [30] *mh
	
	for i := 0; i<30; i++ {
		h[i] = new(mh)
		h[i].hash = fct.Sha([]byte(fmt.Sprintf("h%d",i))).Bytes()
		h[i].time = time.Now().Unix()
		
		if !IsTSValid(h[i].hash, h[i].time) { 
			fmt.Println("Failed Test ",i,"first")
			test.Fail() 
			return
		}
		if IsTSValid(h[i].hash, h[i].time) { 
			fmt.Println("Failed Test ",i,"second")
			test.Fail()
			return
		}
		if !testing.Short() {
			time.Sleep(1*time.Hour)
		}
		for j:=0; j<i; j++ {
			if IsTSValid(h[i].hash, h[i].time) {
				fmt.Println("Failed Test ",i,j,"repeat")
				test.Fail()
				return
			}
		}
		if !testing.Short() {
			fmt.Println("Okay so far:",i)
		}		
	}
			
}