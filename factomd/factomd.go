// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	//	"net"
	//	"net/http"
	//	_ "net/http/pprof"
	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/factomclient"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/restapi"
	"github.com/FactomProject/FactomCode/util"
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
	db              database.Db                          // database
	inMsgQueue      = make(chan factomwire.Message, 100) //incoming message queue for factom application messages
	outMsgQueue     = make(chan factomwire.Message, 100) //outgoing message queue for factom application messages
	inRpcQueue      = make(chan factomwire.Message, 100) //incoming message queue for factom application messages
	federatedid     string
)

// winServiceMain is only invoked on Windows.  It detects when btcd is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, error)

// factomdMain is the real main function for btcd.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the server once it is setup so it can gracefully stop it when
// requested from the service control manager.
func factomdMain(serverChan chan<- *server) error {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	/*	tcfg, _, err := loadConfig()
		if err != nil {
			return err
		}
		cfg = tcfg
		defer backendLog.Flush()

		// Show version at startup.
		fmt.Printf("Version %s\n", version())

		// Enable http profiling server if requested.
		if cfg.Profile != "" {
			go func() {
				listenAddr := net.JoinHostPort("", cfg.Profile)
				fmt.Printf("Profile server listening on %s\n", listenAddr)
				profileRedirect := http.RedirectHandler("/debug/pprof",
					http.StatusSeeOther)
				http.Handle("/", profileRedirect)
				fmt.Printf("%v\n", http.ListenAndServe(listenAddr, nil))
			}()
		}

		// Write cpu profile if requested.
		if cfg.CPUProfile != "" {
			f, err := os.Create(cfg.CPUProfile)
			if err != nil {
				fmt.Printf("Unable to create cpu profile: %v\n", err)
				return err
			}
			pprof.StartCPUProfile(f)
			defer f.Close()
			defer pprof.StopCPUProfile()
		}

		// Perform upgrades to btcd as new versions require it.
		if err := doUpgrades(); err != nil {
			fmt.Printf("%v\n", err)
			return err
		}

		// Load the block database.
		db, err := loadBlockDB()
		if err != nil {
			fmt.Printf("%v\n", err)
			return err
		}
		defer db.Close()

		// Ensure the database is sync'd and closed on Ctrl+C.
		addInterruptHandler(func() {
			fmt.Printf("Gracefully shutting down the database...")
			db.RollbackClose()
		})
	*/

	cfg_Listeners := []string{"0.0.0.0:4012"}

	// Create server and start it.
	server, err := newServer(cfg_Listeners, activeNetParams.Params)
	if err != nil {
		// TODO(oga) this logging could do with some beautifying.
		fmt.Printf("Unable to start server on %v: %v\n",
			cfg_Listeners, err)
		return err
	}
	addInterruptHandler(func() {
		fmt.Printf("Gracefully shutting down the server...")
		server.Stop()
		server.WaitForShutdown()
	})
	server.Start()
	if serverChan != nil {
		serverChan <- server
	}

	// Write outgoing factom messages into P2P network
	go func() {
		for msg := range outMsgQueue {
			server.BroadcastMessage(msg)
			/*			peerInfoResults := server.PeerInfo()
						for peerInfo := range peerInfoResults{
							fmt.Printf("PeerInfo:%+v", peerInfo)

						}*/
		}
	}()

	go func() {
		for msg := range inRpcQueue {
			fmt.Printf("in range inRpcQueue, msg:%+v\n", msg)
			switch msg.Command() {
			case factomwire.CmdTx:
				// inMsgQueue <- msg			for testing
				server.blockManager.QueueTx(msg.(*factomwire.MsgTx), nil)
			case factomwire.CmdConfirmation:
				server.blockManager.QueueConf(msg.(*factomwire.MsgConfirmation), nil)

			default:
				inMsgQueue <- msg
				outMsgQueue <- msg
			}
		}
	}()

	// Monitor for graceful server shutdown and signal the main goroutine
	// when done.  This is done in a separate goroutine rather than waiting
	// directly so the main goroutine can be signaled for shutdown by either
	// a graceful shutdown or from the main interrupt handler.  This is
	// necessary since the main goroutine must be kept running long enough
	// for the interrupt handler goroutine to finish.
	go func() {
		server.WaitForShutdown()
		fmt.Println("Server shutdown complete")
		shutdownChannel <- struct{}{}
	}()

	// Wait for shutdown signal from either a graceful server stop or from
	// the interrupt handler.
	<-shutdownChannel
	fmt.Println("Shutdown complete")
	return nil
}

func main() {

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
	util.Trace()

	// Work around defer not working after os.Exit()
	if err := factomdMain(nil); err != nil {
		os.Exit(1)
	}

	util.Trace()
}

func init() {

	util.Trace()

	// Load configuration file and send settings to components
	loadConfigurations()

	// Initialize db
	initDB()

	// Start the processor module
	go restapi.Start_Processor(db, inMsgQueue, outMsgQueue)

	// Start the RPC server module in a separate go-routine
	go factomclient.Start_Rpcserver(db, inRpcQueue)

	util.Trace()
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
		factomclient.LoadConfigurations(cfg)
	}

	util.Trace()
	fmt.Println("CHECK cfg= ", cfg)
	util.Trace()
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
