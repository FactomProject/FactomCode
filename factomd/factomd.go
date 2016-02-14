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
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd"
	"github.com/FactomProject/btcd/limits"
	"github.com/FactomProject/factoid/state/stateinit"
)

var (
	_   = fmt.Print
	cfg *util.FactomdConfig
	db  database.Db
)

// winServiceMain is only invoked on Windows.  It detects when btcd is running
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
	//btcd.InitProcessor(db, cfg)

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
	//go btcd.StartProcessor() //, inMsgQueue, outMsgQueue, inCtlMsgQueue, outCtlMsgQueue)

	// Start the wsapi server module in a separate go-routine
	//wsapi.Start(db, inMsgQueue)
	/*
		// wait till the initialization is complete in processor
		hash, err := db.FetchDBHashByHeight(0)
		fmt.Printf("hash:  %s\n", hash)
		if err != nil {
			fmt.Println(err.Error())
		}
		latestDirBlockHash, height, _ := db.FetchBlockHeightCache()
		fmt.Printf("latestDirBlockHash: %d, %s\n", height, latestDirBlockHash)
		if hash != nil {
			for true {
				latestDirBlockHash, height, _ := db.FetchBlockHeightCache()
				fmt.Printf("latestDirBlockHash: %d, %s\n", height, latestDirBlockHash)
				if latestDirBlockHash == nil {
					fmt.Println("Waiting for the processor to be initialized...")
					time.Sleep(2 * time.Second)
				} else {
					break
				}
			}
		}

			if len(os.Args) >= 2 {
				if os.Args[1] == "initializeonly" {
					time.Sleep(time.Second)
					fmt.Println("Initializing only.")
					os.Exit(0)
				}
			} else {
				fmt.Println("\n'factomd initializeonly' will do just that.  Initialize and stop.")
			}
	*/
	// Start the factoid (btcd) component and P2P component
	btcd.StartBtcd(db)
	return nil
}

// Load settings from configuration file: factomd.conf
func loadConfigurations() {
	cfg = util.ReadConfig()
	cp.CP.SetPort(cfg.Controlpanel.Port)
	btcd.LoadConfigurations(cfg)
}

// Initialize the level db and share it with other components
func initDB() {
	//init factoid_bolt db
	fmt.Println("boltDBpath:", cfg.App.BoltDBPath)
	common.FactoidState = stateinit.NewFactoidState(cfg.App.BoltDBPath + "factoid_bolt.db")
	var err error
	db, err = ldb.OpenLevelDB(cfg.App.LdbPath, false)
	if err != nil {
		fmt.Printf("err opening db: %v\n", err)
	}

	if db == nil {
		fmt.Println("Creating new db ...")
		db, err = ldb.OpenLevelDB(cfg.App.LdbPath, true)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("Database started from: " + cfg.App.LdbPath)
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
