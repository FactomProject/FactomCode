// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomd

import (
	"fmt"
	//	"net"
	//	"net/http"
	//	_ "net/http/pprof"
	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/restapi"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wsapi"
	"github.com/FactomProject/btcd/wire"
	"log"
	"os"
	"runtime"
	//	"runtime/pprof"

	//	"github.com/FactomProject/FactomCode/btcd/limits"
)

var (
	cfg             *util.FactomdConfig
	shutdownChannel = make(chan struct{})
	ldbpath         = "/tmp/ldb9"
	db              database.Db                    // database
	InMsgQueue      = make(chan wire.FtmInternalMsg, 100) //incoming message queue for factom application messages
	OutMsgQueue     = make(chan wire.FtmInternalMsg, 100) //outgoing message queue for factom application messages
	inRpcQueue      = make(chan wire.Message, 100) //incoming message queue for factom application messages
	federatedid     string
)

// trying out some flags to optionally disable old BTC functionality ... WIP
var FactomOverride struct {
	//	TxIgnoreMissingParents bool
	temp1                     bool
	TxOrphansInsteadOfMempool bool // allow orphans for block creation
	BlockDisableChecks        bool
}

// winServiceMain is only invoked on Windows.  It detects when btcd is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, error)

func Factomd_main() {

	util.Trace()
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Up some limits.
	//	if err := limits.SetLimits(); err != nil {
	//		os.Exit(1)
	//	}

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

	/*
		// Work around defer not working after os.Exit()
		if err := factomdMain(nil); err != nil {
			os.Exit(1)
		}
	*/

}

func Factomd_init() {

	util.Trace()

	// Load configuration file and send settings to components
	loadConfigurations()

	// Initialize db
	initDB()

	// Start the processor module
	go restapi.Start_Processor(db, InMsgQueue, OutMsgQueue)

	// Start the wsapi server module in a separate go-routine
	wsapi.Start(db, inRpcQueue)

	defer wsapi.Stop()
}

// Load settings from configuration file: factomd.conf
func loadConfigurations() {
	tcfg := util.FactomdConfig{}
	cfg = &tcfg

	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	err = gcfg.ReadFileInto(cfg, wd+"/factomd.conf")
	if err != nil {
		log.Println(err)
		log.Println("Server starting with default settings...")
	} else {

		ldbpath = cfg.App.LdbPath
		federatedid = cfg.App.FederatedId
		restapi.LoadConfigurations(cfg)
	}

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
