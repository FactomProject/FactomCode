// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/FactomProject/FactomCode/wallet"
)

func main() {
	fmt.Println("Address: ", wallet.FactoidAddress())
}
