package factomclient

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/FactomProject/FactomCode/factomapi"
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

var server = web.NewServer()

func serve_init() {

	server.Post(`/v1/submitentry/?`, handleSubmitEntry)
	server.Post(`/v1/submitchain/?`, handleSubmitChain)
	server.Post(`/v1/buycredit/?`, handleBuyCreditPost)
	server.Post(`/v1/creditbalance/?`, handleGetCreditBalancePost)
	server.Post(`/v1/addentry/?`, handleSubmitEntry2)       // Needs to be removed later??
	server.Post(`/v1/getfilelist/?`, handleGetFileListPost) // to be replaced by DHT
	server.Post(`/v1/getfile/?`, handleGetFilePost)         // to be replaced by DHT

	server.Get(`/v1/creditbalance/?`, handleGetCreditBalancePost)
	server.Get(`/v1/buycredit/?`, handleBuyCreditPost)

	server.Get(`/v1/dblocksbyrange/([^/]+)(?:/([^/]+))?`, handleDBlocksByRange)
	server.Get(`/v1/dblock/([^/]+)(?)`, handleDBlockByHash)
	server.Get(`/v1/eblock/([^/]+)(?)`, handleEBlockByHash)
	server.Get(`/v1/eblockbymr/([^/]+)(?)`, handleEBlockByMR)
	server.Get(`/v1/entry/([^/]+)(?)`, handleEntryByHash)

}

func handleSubmitEntry(ctx *web.Context) {
	// convert a json post to a factom.Entry then submit the entry to factom
	fmt.Fprintln(ctx, "Entry Submitted")

	switch ctx.Params["format"] {
	case "json":
		entry := new(notaryapi.Entry)
		reader := gocoding.ReadBytes([]byte(ctx.Params["entry"]))
		err := factomapi.SafeUnmarshal(reader, entry)
		if err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err.Error())
		}

		if err := factomapi.CommitEntry(entry); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err.Error())
		}

		time.Sleep(1 * time.Second)
		if err := factomapi.RevealEntry(entry); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err.Error())
		}
		fmt.Fprintln(ctx, "Entry Submitted")
	default:
		ctx.WriteHeader(403)
	}
}

func handleSubmitEntry2(ctx *web.Context) {
	// convert a json post to a factom.Entry then submit the entry to factom
	fmt.Fprintln(ctx, "Entry Submitted")

	switch ctx.Params["format"] {
	case "json":
		j := []byte(ctx.Params["entry"])
		e := new(factomapi.Entry)
		e.UnmarshalJSON(j)
		if err := factomapi.CommitEntry2(e); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err)
		}
		time.Sleep(1 * time.Second)
		if err := factomapi.RevealEntry2(e); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err)
		}
		fmt.Fprintln(ctx, "Entry Submitted")
	default:
		ctx.WriteHeader(403)
	}
}

func handleSubmitChain(ctx *web.Context) {

	// convert a json post to a factomapi.Chain then submit the entry to factomapi
	switch ctx.Params["format"] {
	case "json":
		reader := gocoding.ReadBytes([]byte(ctx.Params["chain"]))
		c := new(notaryapi.EChain)
		factomapi.SafeUnmarshal(reader, c)

		c.GenerateIDFromName()
		if c.FirstEntry == nil {
			fmt.Fprintln(ctx,
				"The first entry is required for submitting the chain:")
			return
		} else {
			c.FirstEntry.ChainID = *c.ChainID
		}

		fmt.Println("c.ChainID:", c.ChainID.String())

		if err := factomapi.CommitChain(c); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the chain:", err)
		}

		time.Sleep(1 * time.Second) //?? do we need to queue them up and look for the confirmation

		if err := factomapi.RevealChain(c); err != nil {
			fmt.Println(err.Error())
			fmt.Fprintln(ctx,
				"there was a problem with submitting the chain:", err)
		}

		fmt.Fprintln(ctx, "Chain Submitted")
	default:
		ctx.WriteHeader(403)
	}
}

func handleBuyCreditPost(ctx *web.Context) {
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

	fmt.Println("handleBuyCreditPost using pubkey: ", ecPubKey, " requested", ctx.Params["to"])

	factoid, _ := strconv.ParseFloat(ctx.Params["value"], 10)
	value := uint64(factoid * 1000000000)
	err := factomapi.BuyEntryCredit(1, ecPubKey, nil, value, 0, nil)

	if err != nil {
		abortMessage = fmt.Sprint("An error occured while submitting the buycredit request: ", err.Error())
		return
	} else {
		fmt.Fprintln(ctx, "MsgGetCredit Submitted")
	}

}
func handleGetCreditBalancePost(ctx *web.Context) {
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
		ecPubKey.Bytes, _ = hex.DecodeString(ctx.Params["pubkey"])
	}

	fmt.Println("handleGetCreditBalancePost using pubkey: ", ecPubKey, " requested", ctx.Params["pubkey"])

	balance, err := factomapi.GetEntryCreditBalance(ecPubKey)

	ecBalance := new(notaryapi.ECBalance)
	ecBalance.Credits = balance
	ecBalance.PublicKey = ecPubKey

	fmt.Println("Balance for pubkey ", ctx.Params["pubkey"], " is: ", balance)

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, ecBalance)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request ")
		return
	}
}

func handleGetFileListPost(ctx *web.Context) {
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
		return
	}
}

func handleGetFilePost(ctx *web.Context) {

	fileKey := ctx.Params["filekey"]
	filename := restapi.ServerDataFileMap[fileKey]
	http.ServeFile(ctx.ResponseWriter, ctx.Request, dataStorePath+"csv/"+filename)

}

func handleDBlocksByRange(ctx *web.Context, fromHeightStr string, toHeightStr string) {
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
		return
	}
	toBlockHeight, err := strconv.Atoi(toHeightStr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad toBlockHeight")
		return
	}

	dBlocks, err := factomapi.GetDirectoryBloks(uint64(fromBlockHeight), uint64(toBlockHeight))
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlocks)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		return
	}

}

func handleDBlockByHash(ctx *web.Context, hashStr string) {
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
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlock)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request ")
		return
	}

}

func handleEBlockByHash(ctx *web.Context, hashStr string) {
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
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		return
	}

}
func handleEBlockByMR(ctx *web.Context, mrStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	fmt.Println("mrstr:", mrStr)
	newstr, _ := url.QueryUnescape(mrStr)
	fmt.Println("newstr:", newstr)
	eBlock, err := factomapi.GetEntryBlokByMRStr(newstr)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		return
	}

}

func handleEntryByHash(ctx *web.Context, hashStr string) {
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
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, entry)
	if err != nil {
		httpcode = 400
		buf.WriteString("Bad request")
		return
	}
}
