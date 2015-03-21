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
	cfg               = util.ReadConfig().Wsapi
	portNumber        = cfg.PortNumber
	applicationName   = cfg.ApplicationName
	dataStorePath     = "/tmp/store/seed/csv"
	refreshInSeconds  = cfg.RefreshInSeconds
)

var server = web.NewServer()

// Start runs the wsapi server which 
func Start(db database.Db, outMsgQ chan<- wire.Message) {
	factomapi.SetDB(db)
	factomapi.SetOutMsgQueue(outMsgQ)

	wsLog.Debug("Setting handlers")
	server.Post(`/v1/buycredit/?`, handleBuyCredit)
	server.Post(`/v1/creditbalance/?`, handleCreditBalance)
//	server.Post(`/v1/factoidtx/?`, handleFactoidTx)
	server.Post(`/v1/submitchain/?`, handleSubmitChain)
	server.Post(`/v1/submitentry/?`, handleSubmitEntry)

	server.Get(`/v1/blockheight/?`, handleBlockHeight)
	server.Get(`/v1/buycredit/?`, handleBuyCredit)
	server.Get(`/v1/creditbalance/?`, handleCreditBalance)
	server.Get(`/v1/dblock/([^/]+)(?)`, handleDBlockByHash)
	server.Get(`/v1/dblocksbyrange/([^/]+)(?:/([^/]+))?`, handleDBlocksByRange)
	server.Get(`/v1/eblock/([^/]+)(?)`, handleEBlockByHash)
	server.Get(`/v1/eblockbymr/([^/]+)(?)`, handleEBlockByMR)
	server.Get(`/v1/entry/([^/]+)(?)`, handleEntryByHash)
//	server.Get(`/v1/factoidtx/?`, handleFactoidTx)

	wsLog.Info("Starting server")
	go server.Run("localhost:" + strconv.Itoa(portNumber))
}

func Stop() {
	server.Close()
}
