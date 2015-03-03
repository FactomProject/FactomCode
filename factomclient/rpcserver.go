package factomclient

import (
	"fmt"
	"github.com/FactomProject/dynrsrc"
	//	"log"
	//	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"
	//	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/util"
	"time"
)

var (
	portNumber       int = 8088
	applicationName      = "factom/client"
	serverAddr           = "localhost:8083"
	//ldbpath              = "/tmp/factomclient/ldb9"
	dataStorePath        = "/tmp/store/seed/csv"
	refreshInSeconds int = 60

	db database.Db // database

	//Map to store imported csv files
	clientDataFileMap map[string]string
)

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init_rpcserver() {
	log := rpcLog
	log.Debug("dynrsrc.Start")
	err := dynrsrc.Start(watchError, readError)
	if err != nil {
		log.Alert(err)
	}

	log.Info("Starting server")
	serve_init()
	//initClientDataFileMap()

	// Import data related to new factom blocks created on server
	ticker := time.NewTicker(time.Second * time.Duration(refreshInSeconds))
	go func() {
		for _ = range ticker.C {
			//downloadAndImportDbRecords()
		}
	}()
}

func Start_Rpcserver(ldb database.Db, outMsgQ chan<- factomwire.Message) {
	db = ldb
	factomapi.SetDB(db)
	factomapi.SetOutMsgQueue(outMsgQ)

	rpcLog.Info("Starting rpcserver")
	init_rpcserver()

	defer func() {
		dynrsrc.Stop()
		server.Close()
	}()

	server.Run(fmt.Sprint(":", portNumber))

}

func LoadConfigurations(cfg *util.FactomdConfig) {
	applicationName = cfg.Rpc.ApplicationName
	portNumber = cfg.Rpc.PortNumber
	dataStorePath = cfg.App.DataStorePath
	refreshInSeconds = cfg.Rpc.RefreshInSeconds
}