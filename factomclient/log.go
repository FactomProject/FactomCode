package factomclient

import (
	"github.com/FactomProject/FactomCode/factomlog"
	"os"
)

var (
	logfile, _ = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)
	rpcLog     = factomlog.New(logfile, logLevel, "rpc")
	serverLog  = factomlog.New(logfile, logLevel, "serv")
)
