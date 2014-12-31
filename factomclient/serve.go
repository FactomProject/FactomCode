package main

import (
	"github.com/FactomProject/FactomCode/wallet"
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/FactomProject/gobundle"
	"github.com/FactomProject/gocoding"
	"github.com/hoisie/web"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomapi"
	"net/http"
//	"net/url"
//	"io/ioutil"
	"strconv"
	"sort"
//	"strings"
	"encoding/base64"
	
  
)

var server = web.NewServer()

func serve_init() {
	server.Config.StaticDir = gobundle.DataFile("/static")
	
	
	server.Get(`/(?:home)?`, handleHome)
	server.Get(`/entries/?`, handleEntries)
	server.Get(`/keys/?`, handleKeys)
	server.Get(`/explore((?:/(?:\w|\d)+)+)?`, handleExplore)
	server.Get(`/settings/?`, handleSettings)
	server.Get(`/failed`, handleFailed)
	
	server.Post(`/v1/submitentry/?`, handleEntryPost)
	server.Post(`/v1/addchain/?`, handleChainPost)	
	server.Post(`/v1/buycredit/?`, handleBuyCreditPost)		
	server.Post(`/v1/creditbalance/?`, handleGetCreditBalancePost)			
	
	server.Get(`/v1/dblocksbyrange/([^/]+)(?:/([^/]+))?`, handleDBlocksByRange)
	server.Get(`/v1/dblock/([^/]+)(?)`, handleDBlockByHash)	
	server.Get(`/v1/eblock/([^/]+)(?)`, handleEBlockByHash)	
	server.Get(`/v1/entry/([^/]+)(?)`, handleEntryByHash)	
	
//	server.Get(`/entries/(?:add|\+)`, handleAddEntry)
	server.Get(`/entries/(?:add|\+)`, handleClientEntry)	
	server.Get(`/entries/([^/]+)(?:/([^/]+)(?:/([^/]+))?)?`, handleEntry)
	server.Get(`/keys/(?:add|\+)`, handleAddKey)
	server.Get(`/keys/([^/]+)(?:/([^/]+))?`, handleKey)
	server.Get(`/dblock/([^/]+)(?)`, handleDBlock)	
	server.Get(`/dblock/?`, handleAllDBlocks)		
	server.Get(`/eblock/([^/]+)(?)`, handleEBlock)
	server.Get(`/sentry/([^/]+)(?)`, handleSEntry)	
	server.Get(`/search/?`, handleSearch)
//	server.Get(`/addchain/?`, handleChain)		
	server.Get(`/chains/?`, handleChains)	
	server.Get(`/chains/(?:add|\+)`, handleAddChain)
	server.Get(`/chain/([^/]+)(?)`, handleChain)
} 

func safeWrite(ctx *web.Context, code int, data map[string]interface{}) *notaryapi.Error {
	var buf bytes.Buffer
	
	data["EntryCount"] = getEntryCount()
	data["KeyCount"] = getEntryCount()
	
	err := mainTmpl.Execute(&buf, data)
	if err != nil { return notaryapi.CreateError(notaryapi.ErrorTemplateError, err.Error()) }
	
	ctx.WriteHeader(code)
	ctx.Write(buf.Bytes())
	
	return nil
}

func safeWrite200(ctx *web.Context, data map[string]interface{}) {
	err := safeWrite(ctx, 200, data)
	if err != nil {
		handleError(ctx, err)
	}
}

func handleHome(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Home",
		"ContentTmpl": "home.md",
	})
}

func handleEntries(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Entries",
		"ContentTmpl": "entries.gwp",
		"AddEntry": true,
	})
}

func handleExplore(ctx *web.Context, rest string) {
	
	if rest == "" || rest == "/" {
		handleAllDBlocks(ctx)
		return
	}	
	if rest == "" || rest == "/" {
		safeWrite200(ctx, map[string]interface{} {
			"Title": "Explore",
			"ContentTmpl": "exploreintro.md",
		})
		return
	}
	
	url := fmt.Sprintf(`http://%s/v1/blocks%s?byref=true`, Settings.Server, rest)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorHTTPNewRequestFailure, err.Error())); return }
	
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorHTTPDoRequestFailure, err.Error())); return }
	
	data := &map[string]interface{} {}
	err = factomapi.SafeUnmarshal(gocoding.Read(resp.Body, 1024), data)
	resp.Body.Close()
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorJSONUnmarshal, err.Error())); return }
	
	if num, ok := (*data)["NumEntries"]; ok {
		links := make([]*link, num.(int64))
		for i := int64(0); i < num.(int64); i++ {
			links[i] = &link{fmt.Sprintf(`Entry %d`, i), fmt.Sprintf(`/explore%s/entries/%d`, rest, i)}
		}
		(*data)["Entries"] = links
		delete(*data, "NumEntries")
	}
	
	buf := new(bytes.Buffer); //buf.ReadFrom(resp.Body)
	err = factomapi.SafeMarshalHTML(buf, data)
	if err != nil { handleError(ctx, notaryapi.CreateError(notaryapi.ErrorHTMLMarshal, err.Error())); return }
	
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Explore",
		"ContentTmpl": "explore.md",
		"Data": buf.String(),
	})
}

func handleSettings(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Settings",
		"ContentTmpl": "settings.gwp",
		"Server": Settings.Server,
	})
}

func handleKeys(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Keys",
		"ContentTmpl": "keys.gwp",
		"AddKey": true,
	})
}

func handleFailed(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Failure",
		"ContentTmpl": "failed.md",
		"Message": ctx.Params["message"],
		"Return": ctx.Params["return"],
	})
}

func handleEntryPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		} else if abortReturn != "" {
			ctx.Header().Add("Location", abortReturn)
			ctx.WriteHeader(303)
		}
	}()
	
	entry := new (notaryapi.Entry)
	reader := gocoding.ReadBytes([]byte(ctx.Params["entry"]))
	err := factomapi.SafeUnmarshal(reader, entry)

	err = factomapi.RevealEntry(1, entry)
		
	if err != nil {
		abortMessage = fmt.Sprint("An error occured while submitting the entry (entry may have been accepted by the server but was not locally flagged as such): ", err.Error())
		return
	}
		
}
func handleBuyCreditPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		} 
	}()

	
	ecPubKey := new (notaryapi.Hash)
	if ctx.Params["to"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		ecPubKey.Bytes, _ = base64.StdEncoding.DecodeString(ctx.Params["to"])
	}

	fmt.Println("handleBuyCreditPost using pubkey: ", ecPubKey, " requested",ctx.Params["to"])

	factoid, _ := strconv.ParseFloat(ctx.Params["value"], 10)
	value := uint64(factoid*1000000000)
	err := factomapi.BuyEntryCredit(1, ecPubKey, nil, value, 0, nil)

		
	if err != nil {
		abortMessage = fmt.Sprint("An error occured while submitting the buycredit request: ", err.Error())
		return
	}
		 
}
func handleGetCreditBalancePost(ctx *web.Context) {	
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	ecPubKey := new (notaryapi.Hash)
	if ctx.Params["pubkey"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		ecPubKey.Bytes, _ = base64.StdEncoding.DecodeString(ctx.Params["pubkey"])
	}

	fmt.Println("handleGetCreditBalancePost using pubkey: ", ecPubKey, " requested",ctx.Params["pubkey"])
	
	balance, err := factomapi.GetEntryCreditBalance(ecPubKey)
	
	ecBalance := new(notaryapi.ECBalance)
	ecBalance.Credits = balance
	ecBalance.PublicKey = ecPubKey

	fmt.Println("Balance for pubkey ", ctx.Params["pubkey"], " is: ", balance)
	
	// Send back JSON response
	err = factomapi.SafeMarshal(buf, ecBalance)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request ")
		return		
	}			
}
func handleKeysPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()
	
	switch ctx.Params["action"] {
	case "genKey":
		abortReturn = fmt.Sprint("/keys/add")
		
		sid := ctx.Params["algorithm"]
		id, err := strconv.Atoi(sid)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to generate key: error parsing algorithm id: ", err.Error())
			return
		}
		
		key, err := notaryapi.GenerateKeyPair(int8(id), rand.Reader)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to generate key: error generating key: ", err.Error())
			return
		}
		if key == nil {
			abortMessage = fmt.Sprint("Failed to generate key: unsupported algorithm id: ", id)
			return
		}
		
		addKey(key)
		
		ctx.Header().Add("Location", fmt.Sprintf("/keys/%d", Settings.NextKeyID-1))
		ctx.WriteHeader(303)
	}
}
 
func handleChainPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()
	
	fmt.Println("In handlechainPost")	
	chain := new (notaryapi.EChain)
	reader := gocoding.ReadBytes([]byte(ctx.Params["chain"]))
	err := factomapi.SafeUnmarshal(reader, chain)

	err = factomapi.RevealChain(1, chain, nil)
		
	if err != nil {
		abortMessage = fmt.Sprint("An error occured while adding the chain ", err.Error())
		return
	}
	
		
}

func handleSettingsPost(ctx *web.Context) {
	Settings.Server = ctx.Params["server"]
	saveSettings()
	handleSettings(ctx)
}

func handleAddEntry(ctx *web.Context) {
	chains, _ := db.FetchAllChainsByName(nil)
	
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Add Entry",
		"ContentTmpl": "addentry.gwp",
		"chains": chains,
	})
}

func handleClientEntry(ctx *web.Context) {

	chains, _ := db.FetchAllChainsByName(nil)
	entry := NewEntryOfType(EntryDataType(1))
	entry_id := addEntry(entry)	
	fmt.Println("len of chains:%v", len(*chains))
	
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Add Entry",
		"ContentTmpl": "cliententry.gwp",
		"chains": chains,
		"EntryID": entry_id,		
		
	})
} 

func handleEntry(ctx *web.Context, entry_id_str string, action string, action_id_str string) {

	var err error
	var title, error_str, tmpl string
	var entry_id int
	var newdata string
	var externalHashes []notaryapi.Hash
	
	defer func(){
		if action == "submit" {
			tmpl = "entrysubmit.gwp"
		} else {
			tmpl = "entry.gwp"
		}
		
		fentry, ok := entries[entry_id]
		
		if ok{
			newdata = string (fentry.Entry.Data())
			if fentry.Entry.ExtHashes != nil{
				externalHashes = *fentry.Entry.ExtHashes
			}
		}
		
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"ContentTmpl": tmpl,
			"EntryID": entry_id,
			"Error": error_str,
			"Mode": action,
			"ShowEntries": true,
			"externalHashes": externalHashes,
			"newdata": newdata,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	 
	entry_id, err = strconv.Atoi(entry_id_str)
	
	if err != nil  {
		error_str = fmt.Sprintf("Bad entry id: %s", err.Error())
		entry_id = -1
		title = "Entry not found"
		return
	} else {
		title = fmt.Sprint("Entry ", entry_id)
	}
	
	switch action {
	case "", "edit", "sign", "submit":
		return
		
	default:
		error_str = fmt.Sprintf("Unknown action: %s", action)
	}
}

func handleEBlock(ctx *web.Context, eBlockHash string) {
	var err error
	var title, tmpl string
	
	hash,_ := notaryapi.HexToHash(eBlockHash)
	eBlock, _ := db.FetchEBlockByHash(hash)
	ebInfo, _ := db.FetchEBInfoByHash(hash)	
	
	if eBlock == nil || ebInfo == nil  {
		handleMessage(ctx, "Not Found", "Entry block not found for hash: " + eBlockHash)
		return
	}
	

	defer func(){
		tmpl = "eblock.gwp"
		
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"ContentTmpl": tmpl,
			"eBlock": eBlock,	
			"ebHash": eBlockHash,	
			"ebInfo": ebInfo,	
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	
	if err != nil  {
		fmt.Sprintf("Bad block id: %s", err.Error())
		title = "Entry Block not found"
		return
	} else {
		title = fmt.Sprint("Entry Block ", eBlock.Header.BlockID)
	}
	
}

func handleSEntry(ctx *web.Context, entryHash string) {
	var err error
	var title, tmpl string
	
	hash,_ := notaryapi.HexToHash(entryHash)
	entry, _ := db.FetchEntryByHash(hash)
	
	if entry == nil{
		handleMessage(ctx, "Not Found", "Entry not found for hash: " + entryHash)
		return
	}
	
	entryInfo, _ := db.FetchEntryInfoByHash(hash)	
	ebInfo, _ := db.FetchEBInfoByHash(entryInfo.EBHash)
	eblock, _ := db.FetchEBlockByHash(entryInfo.EBHash)
	
	if entryInfo == nil || ebInfo == nil {
		fmt.Sprintf("Bad entry hash: %s", entryHash)
		title = "Entry not found"
		err := new (notaryapi.Error)
		err.Message = "Bad entry hash: " + entryHash
		handleError(ctx, err)
		return
	}
	

	defer func(){
		tmpl = "sentry.gwp"
		
		entryData := string(entry.Data)
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"ContentTmpl": tmpl,
			"entry": entry,	
			"entryInfo": entryInfo,	
			"ebInfo":ebInfo,
			"eblock":eblock,
			"entryData":entryData,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	
	if err != nil  {
		fmt.Sprintf("Bad entry hash: %s", err.Error())
		title = "Entry not found"
		return
	} else {
		title = fmt.Sprint("Entry: ", entryHash)
	}
	
}

func handleAddKey(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Add Key",
		"ContentTmpl": "addkey.gwp",
	})
}


func handleAddChain(ctx *web.Context) {
	safeWrite200(ctx, map[string]interface{} {
		"Title": "Add Chain",
		"ContentTmpl": "addchain.gwp",
	})
}

func handleChains(ctx *web.Context) {
	var title, error_str string	

	chains, _ := db.FetchAllChainsByName(nil)
	//sort.Sort(byBlockID(fbInfoArray))
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "chains.gwp",
			"chains": chains,		
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	
}

func handleDBlocksByRange(ctx *web.Context, fromHeightStr string, toHeightStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	fromBlockHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad fromBlockHeight")
		return
	}
	toBlockHeight, err := strconv.Atoi(toHeightStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad toBlockHeight")
		return		
	}	
	
	dBlocks, err := factomapi.GetDirectoryBloks(uint64(fromBlockHeight), uint64(toBlockHeight))
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}	

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlocks)
	if err != nil{
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
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlock)
	if err != nil{
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
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil{
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
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, entry)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}		
}



func handleKey(ctx *web.Context, key_id_str string, action string) {
	var err error
	var title, error_str string
	var key_id int

	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"ContentTmpl": "key.gwp",
			"KeyID": key_id,
			"Edit": action == "edit",
			"Error": error_str,
			"ShowKeys": true,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	key_id, err = strconv.Atoi(key_id_str)
	
	if err != nil  {
		error_str = fmt.Sprintf("Bad key id: %s", err.Error())
		key_id = -1
		title = "Key not found"
		return
	} else {
		title = fmt.Sprint("Key ", key_id)
	}
	
	switch {
	case action == "" || action == "edit":
		return
	
	default:
		error_str = fmt.Sprintf("Unknown action: %s", action)
	}
}

func handleError(ctx *web.Context, err *notaryapi.Error) {
	var buf bytes.Buffer
	
	r := notaryapi.Marshal(err, "json", &buf, false)
	if r != nil { err = r }
	
	err = safeWrite(ctx, err.HTTPCode, map[string]interface{} {
		"Title": "Error",
		"HTTPCode": err.HTTPCode,
		"Content": buf.String(),
		"ContentTmpl": "error.gwp",
	})
	if err != nil {
		str := fmt.Sprintf(`<!DOCTYPE html>
	<html>
		<head>
			<title>Server Failure</title>
		</head>
		<body>
			Something is seriously broken.
			<pre>%s</pre>
		</body>
	</html>`, err.Error())
		
		ctx.WriteHeader(500)
		ctx.Write([]byte(str))
	}
}


func handleMessage(ctx *web.Context, title string, message string) {
	
	r := safeWrite(ctx,200, map[string]interface{} {
		"Title": title,
		"Content": message,
		"ContentTmpl": "message.gwp",
	})
	if r != nil {
		handleError(ctx, r)
	}	
}
func handleDBlock(ctx *web.Context, hashStr string) {
	
	var title, error_str, dbBatchStatus string	
	hash,_ := notaryapi.HexToHash(hashStr)
	dBlock, _ := db.FetchDBlockByHash(hash)
	dbBatch, _ := db.FetchDBBatchByHash(hash)
	
	for _, dbEntry := range dBlock.DBEntries {
		dbHash, _ := db.FetchEBHashByMR(dbEntry.MerkleRoot)
		dbEntry.SetHash(dbHash.Bytes)
	}
	
	if dbBatch != nil {
		dbBatchStatus = "existing"
	}
	
	if dBlock == nil {
		handleMessage(ctx, "Not Found", "Factom Block not found for hash: " + hashStr)
		return
	}	
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "fblock.gwp",
			"dBlock": dBlock,	
			"dbHash": hashStr,	
			"dbBatch": dbBatch,	
			"dbBatchStatus": dbBatchStatus,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
}

func handleChain(ctx *web.Context, chainIDstr string) {
	
	var title, error_str string	
	chainID,_ := notaryapi.HexToHash(chainIDstr)
	
	chain, _ := db.FetchChainByHash(chainID)
	
	eBlocks, _ := db.FetchAllEBlocksByChain(chainID)
	sort.Sort(byEBlockID(*eBlocks))
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "chain.gwp",
			"eBlocks": eBlocks,		
			"chain": chain,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
}

func handleAllDBlocks(ctx *web.Context) {
	var title, error_str string	

	dBlocks, _ := db.FetchAllDBlocks()
	sort.Sort(byBlockID(dBlocks))
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "fblocks.gwp",
			"dBlocks": dBlocks,		
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
	
}

func handleSearch(ctx *web.Context) {
	var title, error_str string	 

	inputhash := ctx.Params["inputhash"]
	hash := new (notaryapi.Hash)
	hash.Bytes, _ = notaryapi.DecodeBinary(&inputhash)

	switch hashtype := ctx.Params["hashtype"]; hashtype {
	case "entry":
		handleSEntry(ctx, inputhash)

	case "eblock":
		handleEBlock(ctx, inputhash)
			
	case "fblock":
		handleDBlock(ctx, inputhash)
		
	default:
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "search.gwp",	
		})
		if r != nil {
			handleError(ctx, r)
		}	
	}		
	
}

// array sorting implementation - assending
type byBlockID []notaryapi.DBlock
func (f byBlockID) Len() int { 
  return len(f) 
} 
func (f byBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID < f[j].Header.BlockID
} 
func (f byBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 

// array sorting implementation - assending
type byEBlockID []notaryapi.EBlock
func (f byEBlockID) Len() int { 
  return len(f) 
} 
func (f byEBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID < f[j].Header.BlockID
} 
func (f byEBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 
