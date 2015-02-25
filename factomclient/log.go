package factomclient

import (
	"fmt"
	"os"
	"github.com/FactomProject/FactomCode/factomlog"
)

var (
	rpcLog    *factomlog.FLogger
	serverLog *factomlog.FLogger
)

func init() {
	logfile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0660)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	
	rpcLog    = factomlog.New(logfile, logLevel, "rpc")
	serverLog = factomlog.New(logfile, logLevel, "serv")
}