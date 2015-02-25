package factomclient

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/restapi"
	"github.com/FactomProject/FactomCode/wallet"
	"github.com/FactomProject/gocoding"
	"github.com/hoisie/web"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	server = web.NewServer()
)

func serve_init() {
	serverLog.Info("Starting api handlers")
	server.Post(`/v1/submitentry/?`, handleSubmitEntry)
	server.Post(`/v1/submitchain/?`, handleSubmitChain)
	server.Post(`/v1/buycredit/?`, handleBuyCreditPost)
	server.Post(`/v1/creditbalance/?`, handleGetCreditBalance)
	server.Post(`/v1/getfilelist/?`, handleGetFileListPost) // to be replaced by DHT
	server.Post(`/v1/getfile/?`, handleGetFilePost)         // to be replaced by DHT
	server.Post(`/v1/factoidtx/?`, handleFactoidTx)
	//	server.Post(`/v1/bintx/?`, handleBinTx)

	server.Get(`/v1/creditbalance/?`, handleGetCreditBalance)
	server.Get(`/v1/buycredit/?`, handleBuyCreditPost)
	server.Get(`/v1/dblocksbyrange/([^/]+)(?:/([^/]+))?`, handleDBlocksByRange)
	server.Get(`/v1/dblock/([^/]+)(?)`, handleDBlockByHash)
	server.Get(`/v1/eblock/([^/]+)(?)`, handleEBlockByHash)
	server.Get(`/v1/eblockbymr/([^/]+)(?)`, handleEBlockByMR)
	server.Get(`/v1/entry/([^/]+)(?)`, handleEntryByHash)
	server.Get(`/v1/factoidtx/?`, handleFactoidTx)
	//	server.Get(`/v1/bintx/?`, handleBinTx)
}

// handle Submit Entry converts a json post to a factom.Entry then submits the
// entry to factom.
func handleSubmitEntry(ctx *web.Context) {
	log := serverLog
	log.Debug("handleSubmitEntry")

	switch ctx.Params["format"] {
	case "json":
		entry := new(notaryapi.Entry)
		reader := gocoding.ReadBytes([]byte(ctx.Params["entry"]))
		err := factomapi.SafeUnmarshal(reader, entry)
		if err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err.Error())
			log.Error(err)
		}

		if err := factomapi.CommitEntry(entry); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err.Error())
			log.Error(err)
		}

		time.Sleep(1 * time.Second)
		if err := factomapi.RevealEntry(entry); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err.Error())
			log.Error(err)
		}
		fmt.Fprintln(ctx, "Entry Submitted")
	default:
		ctx.WriteHeader(403)
	}
}

// handleSubmitChain converts a json post to a factomapi.Chain then submits the
// entry to factomapi.
func handleSubmitChain(ctx *web.Context) {
	log := serverLog
	log.Debug("handleSubmitChain")
	switch ctx.Params["format"] {
	case "json":
		reader := gocoding.ReadBytes([]byte(ctx.Params["chain"]))
		c := new(notaryapi.EChain)
		factomapi.SafeUnmarshal(reader, c)

		c.GenerateIDFromName()
		if c.FirstEntry == nil {
			fmt.Fprintln(ctx,
				"The first entry is required for submitting the chain:")
			log.Warning("The first entry is required for submitting the chain")
			return
		} else {
			c.FirstEntry.ChainID = *c.ChainID
		}

		log.Debug("c.ChainID:", c.ChainID.String())

		if err := factomapi.CommitChain(c); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the chain:", err)
			log.Error(err)
		}

		time.Sleep(1 * time.Second)

		if err := factomapi.RevealChain(c); err != nil {
			fmt.Println(err.Error())
			fmt.Fprintln(ctx,
				"there was a problem with submitting the chain:", err)
			log.Error(err)
		}

		fmt.Fprintln(ctx, "Chain Submitted")
	default:
		ctx.WriteHeader(403)
	}
}

// handleBuyCreditPost will add entry credites to the specified key. Currently
// the entry credits are given without any factoid transactions occuring.
func handleBuyCreditPost(ctx *web.Context) {
	log := serverLog
	log.Debug("handleBuyCreditPost")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	var abortMessage, abortReturn string

	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()

	ecPubKey := new(notaryapi.Hash)
	if ctx.Params["to"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		ecPubKey.Bytes, _ = hex.DecodeString(ctx.Params["to"])
	}

	log.Info("handleBuyCreditPost using pubkey: ", ecPubKey, " requested", ctx.Params["to"])

	factoid, err := strconv.ParseFloat(ctx.Params["value"], 10)
	if err != nil {
		log.Error(err)
	}
	value := uint64(factoid * 1000000000)
	err = factomapi.BuyEntryCredit(1, ecPubKey, nil, value, 0, nil)
	if err != nil {
		abortMessage = fmt.Sprint("An error occured while submitting the buycredit request: ", err.Error())
		log.Error(err)
		return
	} else {
		fmt.Fprintln(ctx, "MsgGetCredit Submitted")
	}
}

//func handleBinTx(ctx *web.Context) {
//	msg := new(factomwire.MsgTx)
//	msg.Data, err := hex.DecodeString(ctx.Params["tx"])
//	if err != nil {
//		fmt.Println(err)
//	}
//	//factomwire.FactomRelay(msg)
//}

func handleFactoidTx(ctx *web.Context) {
	log := serverLog
	log.Debug("handleFactoidTx")
	n, err := strconv.ParseInt(ctx.Params["amount"], 10, 32)
	if err != nil {
		log.Error(err)
	}
	amt := n

	addr, _, err := factoid.DecodeAddress(ctx.Params["to"])
	if err != nil {
		log.Error(err)
	}

	log.Debug("factoid.NewTxFromInputToAddr")
	txm := factoid.NewTxFromInputToAddr(
		factoid.NewFaucetIn(),
		amt,
		addr)
	ds := wallet.DetachMarshalSign(txm.TxData)
	ss := factoid.NewSingleSignature(ds)
	factoid.AddSingleSigToTxMsg(txm, ss)

	wire := factoid.TxMsgToWire(txm)

	time.Sleep(1 * time.Second)
	if err := factomapi.SubmitFactoidTx(wire); err != nil {
		fmt.Fprintln(ctx,
			"there was a problem submitting the tx:", err.Error())
		log.Error(err)
	}

	log.Debug("MsgTx: ", wire)
}

// handleGetCreditBalance will return the current entry credit balance of the
// spesified pubKey
func handleGetCreditBalance(ctx *web.Context) {
	log := serverLog
	log.Debug("handleGetCreditBalance")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	ecPubKey := new(notaryapi.Hash)
	if ctx.Params["pubkey"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		p, err := hex.DecodeString(ctx.Params["pubkey"])
		if err != nil {
			log.Error(err)
		}
		ecPubKey.Bytes = p
	}

	log.Info("handleGetCreditBalance using pubkey: ", ecPubKey,
		" requested", ctx.Params["pubkey"])

	balance, err := factomapi.GetEntryCreditBalance(ecPubKey)
	if err != nil {
		log.Error(err)
	}

	ecBalance := new(notaryapi.ECBalance)
	ecBalance.Credits = balance
	ecBalance.PublicKey = ecPubKey

	log.Info("Balance for pubkey ", ctx.Params["pubkey"], " is: ", balance)

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, ecBalance)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request ")
		log.Error(err)
		return
	}
}

func handleGetFileListPost(ctx *web.Context) {
	log := serverLog
	log.Debug("handleGetFileListPost")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	// Send back JSON response
	err := factomapi.SafeMarshal(buf, restapi.ServerDataFileMap)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request ")
		log.Error(err)
		return
	}
}

func handleGetFilePost(ctx *web.Context) {
	log := serverLog
	log.Debug("handleGetFilePost")

	fileKey := ctx.Params["filekey"]
	filename := restapi.ServerDataFileMap[fileKey]
	http.ServeFile(ctx.ResponseWriter, ctx.Request, dataStorePath+"csv/"+filename)
}

func handleDBlocksByRange(ctx *web.Context, fromHeightStr string,
	toHeightStr string) {
	log := serverLog
	log.Debug("handleDBlocksByRange")

	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	fromBlockHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad fromBlockHeight")
		log.Error(err)
		return
	}
	toBlockHeight, err := strconv.Atoi(toHeightStr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad toBlockHeight")
		log.Error(err)
		return
	}

	dBlocks, err := factomapi.GetDirectoryBloks(uint64(fromBlockHeight), uint64(toBlockHeight))
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		log.Error(err)
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlocks)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		log.Error(err)
		return
	}
}

func handleDBlockByHash(ctx *web.Context, hashStr string) {
	log := serverLog
	log.Debug("handleDBlockByHash")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	dBlock, err := factomapi.GetDirectoryBlokByHashStr(hashStr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad Request")
		log.Error(err)
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlock)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request ")
		log.Error(err)
		return
	}
}

func handleEBlockByHash(ctx *web.Context, hashStr string) {
	log := serverLog
	log.Debug("handleEBlockByHash")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	eBlock, err := factomapi.GetEntryBlokByHashStr(hashStr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad Request")
		log.Error(err)
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		log.Error(err)
		return
	}
}

func handleEBlockByMR(ctx *web.Context, mrStr string) {
	log := serverLog
	log.Debug("handleEBlockByMR")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	log.Info("mrstr:", mrStr)
	newstr, _ := url.QueryUnescape(mrStr)
	log.Info("newstr:", newstr)
	eBlock, err := factomapi.GetEntryBlokByMRStr(newstr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad Request")
		log.Error(err)
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		log.Error(err)
		return
	}
}

func handleEntryByHash(ctx *web.Context, hashStr string) {
	log := serverLog
	log.Debug("handleEBlockByMR")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	entry, err := factomapi.GetEntryByHashStr(hashStr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad Request")
		log.Error(err)
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, entry)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		log.Error(err)
		return
	}
}
