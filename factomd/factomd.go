// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/FactomProject/FactomCode/common"
	cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/server"
	"github.com/FactomProject/FactomCode/server/limits"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wire"
	"github.com/FactomProject/FactomCode/wsapi"
	"github.com/FactomProject/factoid/state/stateinit"
)

var (
	_           = fmt.Print
	cfg         *util.FactomdConfig
	db          database.Db
	inMsgQueue  = make(chan wire.FtmInternalMsg, 100) //incoming message queue for factom application messages
	outMsgQueue = make(chan wire.FtmInternalMsg, 100) //outgoing message queue for factom application messages
)

// winServiceMain is only invoked on Windows.  It detects when server is running
// as a service and reacts accordingly.
//var winServiceMain func() (bool, error)

func main() {
	ftmdLog.Info("//////////////////////// Copyright 2015 Factom Foundation")
	ftmdLog.Info("//////////////////////// Use of this source code is governed by the MIT")
	ftmdLog.Info("//////////////////////// license that can be found in the LICENSE file.")
	ftmdLog.Warning("Go compiler version:", runtime.Version())
	fmt.Println("Go compiler version:", runtime.Version())

	// Load configuration file and send settings to components
	loadConfigurations()
	
	cp.CP.AddUpdate("gocompiler",
		"system",
		fmt.Sprintln("Go compiler version: ", runtime.Version()),
		"",
		0)
	cp.CP.AddUpdate("copyright",
		"system",
		"Legal",
		"Copyright 2015 Factom Foundation\n"+
			"Use of this source code is governed by the MIT\n"+
			"license that can be found in the LICENSE file.",
		0)
	
	if !isCompilerVersionOK() {
		for i := 0; i < 30; i++ {
			fmt.Println("!!! !!! !!! ERROR: unsupported compiler version !!! !!! !!!")
		}
		time.Sleep(time.Second)
		os.Exit(1)
	}

	// create the $home/.factom directory if it does not exist
	os.Mkdir(cfg.App.HomeDir, 0755)

	// Initialize db
	initDB()
	server.InitProcessor(db)

	// Use all processor cores.
	//runtime.GOMAXPROCS(runtime.NumCPU())

	//Up some limits.
	if err := limits.SetLimits(); err != nil {
		os.Exit(1)
	}

	// Start the wsapi server module in a separate go-routine
	wsapi.Start(db, inMsgQueue)

	server.StartBtcd(db, inMsgQueue, outMsgQueue)
}

// Load settings from configuration file: factomd.conf
func loadConfigurations() {
	cfg = util.ReadConfig()
	cp.CP.SetPort(cfg.Controlpanel.Port)
	server.LoadConfigurations(cfg)
}

// Initialize the level db and share it with other components
func initDB() {
	//init factoid_bolt db
	fmt.Println("boltDBpath:", cfg.App.BoltDBPath)
	common.FactoidState = stateinit.NewFactoidState(cfg.App.BoltDBPath + "factoid_bolt.db")
	var err error
	db, err = ldb.OpenLevelDB(cfg.App.LdbPath, false)
	if err != nil {
		ftmdLog.Errorf("err opening db: %v\n", err)
	}

	if db == nil {
		ftmdLog.Info("Creating new db ...")
		db, err = ldb.OpenLevelDB(cfg.App.LdbPath, true)
		if err != nil {
			panic(err)
		}
	}
	ftmdLog.Info("Database started from: " + cfg.App.LdbPath)
}

func isCompilerVersionOK() bool {
	goodenough := false
	if strings.Contains(runtime.Version(), "1.4") {
		goodenough = true
	}
	if strings.Contains(runtime.Version(), "1.5") {
		goodenough = true
	}
	if strings.Contains(runtime.Version(), "1.6") {
		goodenough = true
	}
	return goodenough
}
