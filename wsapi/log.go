// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wsapi

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
	rpcLog    = factomlog.New(logfile, logLevel, "RPC")
	serverLog = factomlog.New(logfile, logLevel, "SERV")
	wsLog     = factomlog.New(logfile, logLevel, "WSAPI")
)
