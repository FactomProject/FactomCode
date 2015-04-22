// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wsapi"
	"github.com/FactomProject/btcd"
	"github.com/FactomProject/btcd/limits"
	"github.com/FactomProject/btcd/wire"
	"log"
	"os"
	"runtime"
)

var (
	cfg             *util.FactomdConfig
	shutdownChannel = make(chan struct{})
	ldbpath         = "/tmp/ldb9"
	db              database.Db                           // database
	inMsgQueue      = make(chan wire.FtmInternalMsg, 100) //incoming message queue for factom application messages
	outMsgQueue     = make(chan wire.FtmInternalMsg, 100) //outgoing message queue for factom application messages
	inCtlMsgQueue   = make(chan wire.FtmInternalMsg, 100) //incoming message queue for factom application messages
	outCtlMsgQueue  = make(chan wire.FtmInternalMsg, 100) //outgoing message queue for factom application messages
	//	inRpcQueue      = make(chan wire.Message, 100) //incoming message queue for factom application messages
	federatedid string
)

// winServiceMain is only invoked on Windows.  It detects when btcd is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, error)

func main() {
	fmt.Println("//////////////////////// Copyright 2015 Factom Foundation")
	fmt.Println("//////////////////////// Use of this source code is governed by the MIT")
	fmt.Println("//////////////////////// license that can be found in the LICENSE file.")

	util.Trace()
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	//Up some limits.
	if err := limits.SetLimits(); err != nil {
		os.Exit(1)
	}

	// Call serviceMain on Windows to handle running as a service.  When
	// the return isService flag is true, exit now since we ran as a
	// service.  Otherwise, just fall through to normal operation.
/*
	if runtime.GOOS == "windows" {
		isService, err := winServiceMain()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if isService {
			os.Exit(0)
		}
	}
*/
	// Work around defer not working after os.Exit()
	if err := factomdMain(); err != nil {
		os.Exit(1)
	}

}

func init() {

	util.Trace()

	// Load configuration file and send settings to components
	loadConfigurations()

	// Initialize db
	initDB()

}

func factomdMain() error {

	// Start the processor module
	go btcd.Start_Processor(db, inMsgQueue, outMsgQueue, inCtlMsgQueue, outCtlMsgQueue)

	// Start the wsapi server module in a separate go-routine
	go wsapi.Start(db, inMsgQueue)

	// Start the factoid (btcd) component and P2P component
	btcd.Start_btcd(inMsgQueue, outMsgQueue, inCtlMsgQueue, outCtlMsgQueue)

	return nil
}

// Load settings from configuration file: factomd.conf
func loadConfigurations() {

	cfg = util.ReadConfig()

	ldbpath = cfg.App.LdbPath
	federatedid = cfg.App.FederatedId
	btcd.LoadConfigurations(cfg)

	fmt.Println("CHECK cfg= ", cfg)
}

// Initialize the level db and share it with other components
func initDB() {

	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)

	if err != nil {
		log.Printf("err opening db: %v\n", err)

	}

	if db == nil {
		log.Println("Creating new db ...")
		db, err = ldb.OpenLevelDB(ldbpath, true)

		if err != nil {
			panic(err)
		}
	}
	log.Println("Database started from: " + ldbpath)

}
