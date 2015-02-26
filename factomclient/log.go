package factomclient

import (
	"os"

	"github.com/FactomProject/FactomCode/factomlog"
)

var logfile, _ = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)

var (
	rpcLog     = factomlog.New(logfile, logLevel, "rpc")
	serverLog  = factomlog.New(logfile, logLevel, "serv")
)
