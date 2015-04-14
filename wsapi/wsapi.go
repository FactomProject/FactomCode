// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wsapi

import (
	"strconv"

	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/FactomCode/util"
	"github.com/hoisie/web"
)

var (
	cfg              = util.ReadConfig().Wsapi
	portNumber       = cfg.PortNumber
	applicationName  = cfg.ApplicationName
	dataStorePath    = "/tmp/store/seed/csv"
	refreshInSeconds = cfg.RefreshInSeconds
)

var server = web.NewServer()

// Start runs the wsapi server which
func Start(db database.Db, inMsgQ chan<- wire.FtmInternalMsg) {
	factomapi.SetDB(db)
	factomapi.SetInMsgQueue(inMsgQ)

	wsLog.Debug("Setting handlers")
	server.Post(`/v1/buycredit/?`, handleBuyCredit)
	server.Post(`/v1/creditbalance/?`, handleCreditBalance)
	server.Post(`/v1/submitchain/?`, handleSubmitChain)
	server.Post(`/v1/submitentry/?`, handleSubmitEntry)

	server.Get(`/v1/dblockheight/?`, handleBlockHeight)
	server.Get(`/v1/blockheight/?`, handleBlockHeight)
	server.Get(`/v1/buycredit/?`, handleBuyCredit)
	server.Get(`/v1/chain/([^/]+)(?)`, handleChainByHash)
	server.Get(`/v1/chains/?`, handleChains)
	server.Get(`/v1/creditbalance/?`, handleCreditBalance)
	server.Get(`/v1/dblock/([^/]+)(?)`, handleDBlockByHash)
	server.Get(`/v1/dbinfo/([^/]+)(?)`, handleDBInfoByHash)
	server.Get(`/v1/dblocksbyrange/([^/]+)(?:/([^/]+))?`, handleDBlocksByRange)
	server.Get(`/v1/eblock/([^/]+)(?)`, handleEBlockByMR)
	server.Get(`/v1/eblockbyhash/([^/]+)(?)`, handleEBlockByHash)
	server.Get(`/v1/eblockbymr/([^/]+)(?)`, handleEBlockByMR)
	server.Get(`/v1/entry/([^/]+)(?)`, handleEntryByHash)
	server.Get(`/v1/entriesbyeid/([^/]+)(?)`, handleEntriesByExtID)

	wsLog.Info("Starting server")
	go server.Run("localhost:" + strconv.Itoa(portNumber))
}

func Stop() {
	server.Close()
}
