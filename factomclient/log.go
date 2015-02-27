package factomclient

import (
	"os"

	"github.com/FactomProject/FactomCode/factomlog"
	"github.com/FactomProject/FactomCode/util"
)

var (
	logcfg     = util.ReadConfig().Log
	logPath    = logcfg.LogPath
	logLevel   = logcfg.LogLevel
	logfile, _ = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)
)

// setup subsystem loggers
var (
	rpcLog     = factomlog.New(logfile, logLevel, "rpc")
	serverLog  = factomlog.New(logfile, logLevel, "serv")
)
