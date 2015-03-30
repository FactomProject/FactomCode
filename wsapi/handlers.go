package wsapi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/wallet"
	"github.com/FactomProject/gocoding"
	"github.com/hoisie/web"
)

func handleBlockHeight(ctx *web.Context) {
	log := serverLog
	log.Debug("handleBlockHeight")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	height, err := factomapi.GetBlokHeight()
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad Request")
		log.Error(err)
		return
	}
	
	fmt.Fprint(buf, height)	
}

// handleBuyCredit will add entry credites to the specified key. Currently the
// entry credits are given without any factoid transactions occuring.
func handleBuyCredit(ctx *web.Context) {
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
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=",
				abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()

	ecPubKey := new(common.Hash)
	if ctx.Params["to"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		ecPubKey.Bytes, _ = hex.DecodeString(ctx.Params["to"])
	}

	log.Info("handleBuyCreditPost using pubkey: ", ecPubKey, " requested",
		ctx.Params["to"])

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

// handleCreditBalance will return the current entry credit balance of the
// spesified pubKey
func handleCreditBalance(ctx *web.Context) {
	log := serverLog
	log.Debug("handleGetCreditBalance")
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()

	ecPubKey := new(common.Hash)
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

	ecBalance := new(common.ECBalance)
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

// handleDBlockByHash will take a directory block hash and return the directory
// block information in json format.
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

// handleDBlockByRange will get a block height range and return the information
// for all of the directory blocks within the range in json format.
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

	dBlocks, err := factomapi.GetDirectoryBloks(uint64(fromBlockHeight),
		uint64(toBlockHeight))
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		log.Error(err)
		return
	}

	// Send back JSON response
	for _, block := range dBlocks {
		err = factomapi.SafeMarshal(buf, block)
		if err != nil {
			httpcode = 400
			buf.WriteString("Bad request")
			log.Error(err)
			return
		}
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

//func handleFactoidTx(ctx *web.Context) {
//	log := serverLog
//	log.Debug("handleFactoidTx")
//	n, err := strconv.ParseInt(ctx.Params["amount"], 10, 32)
//	if err != nil {
//		log.Error(err)
//	}
//	amt := n
//	var toaddress string
//	if ctx.Params["to"] == "wallet" {
//		toaddress = wallet.FactoidAddress()
//	} else {
//		toaddress = ctx.Params["to"]
//	}
//
//	addr, _, err := factoid.DecodeAddress(toaddress)
//	if err != nil {
//		log.Error(err)
//	}
//
//	log.Debug("factoid.NewTxFromInputToAddr")
//	txm := factoid.NewTxFromInputToAddr(
//		factoid.NewFaucetIn(),
//		amt,
//		addr)
//	ds := wallet.DetachMarshalSign(txm.TxData)
//	ss := factoid.NewSingleSignature(ds)
//	factoid.AddSingleSigToTxMsg(txm, ss)
//
//	wire := factoid.TxMsgToWire(txm)
//
//	time.Sleep(1 * time.Second)
//	if err := factomapi.SubmitFactoidTx(wire); err != nil {
//		fmt.Fprintln(ctx,
//			"there was a problem submitting the tx:", err.Error())
//		log.Error(err)
//	}
//
//	log.Debug("MsgTx: ", wire)
//}

// handleSubmitChain converts a json post to a factomapi.Chain then submits the
// entry to factomapi.
func handleSubmitChain(ctx *web.Context) {
	log := serverLog
	log.Debug("handleSubmitChain")
	switch ctx.Params["format"] {
	case "json":
		reader := gocoding.ReadBytes([]byte(ctx.Params["chain"]))
		c := new(common.EChain)
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

// handleSubmitEntry converts a json post to a factom.Entry then submits the
// entry to factom.
func handleSubmitEntry(ctx *web.Context) {
	log := serverLog
	log.Debug("handleSubmitEntry")

	switch ctx.Params["format"] {
	case "json":
		entry := new(common.Entry)
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
