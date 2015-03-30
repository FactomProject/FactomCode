// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/FactomProject/FactomCode/factomd"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd"
)

func realMain() {
	factomd.Factomd_init()
	factomd.Factomd_main()

	btcd.FactomSetupOverrides()
	//	go btcd.test_timer() // block-writing tests timer

	util.Trace()
	btcd.Btcd_main()
}

func main() {
	fmt.Println("//////////////////////// Copyright 2015 Factom Foundation")
	fmt.Println("//////////////////////// Use of this source code is governed by the MIT")
	fmt.Println("//////////////////////// license that can be found in the LICENSE file.")

	realMain()
}
