// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package controlpanel

import (
    "encoding/hex"
    "encoding/binary"
    "fmt"
    "time"
    "strings"
    "strconv"
    "math/rand"
    "testing"
)

var _ = strings.Replace
var _ = strconv.ParseInt
var _ = time.Second
var _ = hex.EncodeToString
var _ = fmt.Printf
var _ = rand.New
var _ = binary.Write

// sets up teststate.go                                         
func TestControlPanel (test *testing.T) {
    if testing.Short() {
       return
    }
    
    CP.SetFactomMode("Server")
    for i:=0; i<10; i++ {
        CP.UpdatePeriodMark(i%10+1)
        CP.UpdateBlockHeight(i/10+1)
        time.Sleep(time.Second*2)
    }
    CP.SetFactomMode("Client")
    for i:=0; i<100; i++ {
        CP.UpdatePeriodMark(i%10+1)
        CP.UpdateBlockHeight(i/10+2)
        time.Sleep(time.Second*2)
    }
    
    
}