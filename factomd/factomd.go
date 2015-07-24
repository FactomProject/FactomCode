// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/process"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wsapi"
	"github.com/FactomProject/btcd"
	"github.com/FactomProject/btcd/limits"
	"github.com/FactomProject/btcd/wire"
	"os"
    "fmt"
	"runtime"
	"time"
)

var (
    _               = fmt.Print
	cfg             *util.FactomdConfig
	shutdownChannel = make(chan struct{})
	ldbpath         = ""
	db              database.Db                           // database
	inMsgQueue      = make(chan wire.FtmInternalMsg, 100) //incoming message queue for factom application messages
	outMsgQueue     = make(chan wire.FtmInternalMsg, 100) //outgoing message queue for factom application messages
	inCtlMsgQueue   = make(chan wire.FtmInternalMsg, 100) //incoming message queue for factom application messages
	outCtlMsgQueue  = make(chan wire.FtmInternalMsg, 100) //outgoing message queue for factom application messages
	//	inRpcQueue      = make(chan wire.Message, 100) //incoming message queue for factom application messages
)

// winServiceMain is only invoked on Windows.  It detects when btcd is running
// as a service and reacts accordingly.
//var winServiceMain func() (bool, error)

func main() {
	ftmdLog.Info("//////////////////////// Copyright 2015 Factom Foundation")
	ftmdLog.Info("//////////////////////// Use of this source code is governed by the MIT")
	ftmdLog.Info("//////////////////////// license that can be found in the LICENSE file.")

	// Load configuration file and send settings to components
	loadConfigurations()

	// Initialize db
	initDB()

	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	//Up some limits.
	if err := limits.SetLimits(); err != nil {
		os.Exit(1)
	}

	// Work around defer not working after os.Exit()
	if err := factomdMain(); err != nil {
		os.Exit(1)
	}

}

func factomdMain() error {

	// Start the processor module
	go process.Start_Processor(db, inMsgQueue, outMsgQueue, inCtlMsgQueue, outCtlMsgQueue)

	// Start the wsapi server module in a separate go-routine
	wsapi.Start(db, inMsgQueue)

	// wait till the initialization is complete in processor - to be improved??
	hash, _ := db.FetchDBHashByHeight(0)
	if hash != nil {
		for true {
			latestDirBlockHash, _, _ := db.FetchBlockHeightCache()
			if latestDirBlockHash == nil {
				ftmdLog.Info("Waiting for the processor to be initialized...")
				time.Sleep(2 * time.Second)
			} else {
				break
			}
		}
	}

	if len(os.Args) >=2 {
        if os.Args[1] == "initializeonly" {
            time.Sleep(time.Second)
            fmt.Println("Initializing only.")
            os.Exit(0)
        }
    }else{
        fmt.Println("\n'factomd initializeonly' will do just that.  Initialize and stop.")
    }
	
	
	// Start the factoid (btcd) component and P2P component
	btcd.Start_btcd(db, inMsgQueue, outMsgQueue, inCtlMsgQueue, outCtlMsgQueue, process.FactomdUser, process.FactomdPass, common.SERVER_NODE != cfg.App.NodeMode)

	return nil
}

// Load settings from configuration file: factomd.conf
func loadConfigurations() {

	cfg = util.ReadConfig()

	ldbpath = cfg.App.LdbPath
	process.LoadConfigurations(cfg)

}

// Initialize the level db and share it with other components
func initDB() {

	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)

	if err != nil {
		ftmdLog.Errorf("err opening db: %v\n", err)

	}

	if db == nil {
		ftmdLog.Info("Creating new db ...")
		db, err = ldb.OpenLevelDB(ldbpath, true)

		if err != nil {
			panic(err)
		}
	}
	ftmdLog.Info("Database started from: " + ldbpath)

}
