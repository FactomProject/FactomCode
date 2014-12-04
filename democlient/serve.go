package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/FactomProject/gobundle"
	"github.com/FactomProject/gocoding"
	"github.com/hoisie/web"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomapi"	
	"net/http"
	"strconv"
	"sort"
	"strings"
 
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
	
	server.Post(`/entries/?`, handleEntriesPost)
	server.Post(`/keys/?`, handleKeysPost)
	server.Post(`/settings/?`, handleSettingsPost)
	server.Post(`/search/?`, handleSearch)	
	server.Post(`/addchain/?`, handleChainPost)	
	
//	server.Get(`/entries/(?:add|\+)`, handleAddEntry)
	server.Get(`/entries/(?:add|\+)`, handleClientEntry)	
	server.Get(`/entries/([^/]+)(?:/([^/]+)(?:/([^/]+))?)?`, handleEntry)
	server.Get(`/keys/(?:add|\+)`, handleAddKey)
	server.Get(`/keys/([^/]+)(?:/([^/]+))?`, handleKey)
	server.Get(`/fblock/([^/]+)(?)`, handleFBlock)	
	server.Get(`/fblock/?`, handleAllFBlocks)		
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
		handleAllFBlocks(ctx)
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
	err = safeUnmarshal(gocoding.Read(resp.Body, 1024), data)
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
	err = safeMarshalHTML(buf, data)
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

func handleEntriesPost(ctx *web.Context) {
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
	
	switch action := ctx.Params["action"]; action {
	case "editDataEntry":
		id := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", id)
		idx, err := strconv.Atoi(id)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(idx) {
			abortMessage = fmt.Sprint("Failed to edit data entry data: bad id: ", id)
			return
		}
		
		entry := getEntry(idx)
		err = entry.DecodeFromString(ctx.Params["data"])
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing data: ", err.Error())
			return
		}
		
		storeEntry(idx)
		
	case "submitEntry":
		id := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", id)
		idx, err := strconv.Atoi(id)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to submit entry: error parsing id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(idx) {
			abortMessage = fmt.Sprint("Failed to submit entry: bad id: ", id)
			return
		}

		
		entry := getEntry(idx)
		
		chainid := ctx.Params["chainid"]
		binaryChainID, err := notaryapi.DecodeBinary(&chainid) 
		if err != nil {
			abortMessage = fmt.Sprint("Invalid chain id: ", chainid)
			return
		} else{
			entry.ChainID = binaryChainID
		}		

		externalHashes :=  make ([][]byte, 0, 5)
		ehash0 := strings.TrimSpace(ctx.Params["ehash0"])
		if len(ehash0)>0{
			externalHashes = append(externalHashes, []byte(ehash0))
		}	
		
		ehash1 := strings.TrimSpace(ctx.Params["ehash1"])
		if len(ehash1)>0{
			externalHashes = append(externalHashes, []byte(ehash1))
		}	
		
		ehash2 := strings.TrimSpace(ctx.Params["ehash2"])
		if len(ehash2)>0{
			externalHashes = append(externalHashes, []byte(ehash2))
		}	
		
		entry.ExtIDs = externalHashes

		err = entry.DecodeFromString(ctx.Params["data"])
		if err != nil {
			abortMessage = fmt.Sprint("Failed to edit data entry data: error parsing data: ", err.Error())
			return
		}		
		
		buf := new(bytes.Buffer)
		err = safeMarshal(buf, entry)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to submit entry: entry could not be marshalled: ", err.Error())
			return
		}
		/*		
		server := fmt.Sprintf(`http://%s/v1`, Settings.Server)
		data := url.Values{}
		
		data.Set("format", "binary")
		binaryEntry,_ := serverEntry.MarshalBinary()
		
		data.Set("entry", notaryapi.EncodeBinary(&binaryEntry))
		
		resp, err := http.PostForm(server, data)		
		var entryHash string
		if body != nil{
			entryHash = notaryapi.EncodeBinary(&body)
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()		
		*/
		serverEntry := new (notaryapi.Entry)
		serverEntry.ChainID.Bytes = entry.ChainID
		serverEntry.ExtIDs = externalHashes
		serverEntry.Data = entry.Data()
		
		err = factomapi.RevealEntry(1, serverEntry)

		if err != nil {
			abortMessage = fmt.Sprint("An error occured while submitting the entry (entry may have been accepted by the server but was not locally flagged as such): ", err.Error())
			return
		}
		
		entryHash, _ := notaryapi.CreateHash(serverEntry)
		
		flagSubmitEntry(idx, entryHash.String())
		
	
	case "rmEntrySig":
		entry_id_str := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", entry_id_str)
		entry_id, err := strconv.Atoi(entry_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to remove entry signature: error parsing entry id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(entry_id) {
			abortMessage = fmt.Sprint("Failed to remove entry signature: bad entry id: ", entry_id_str)
			return
		}
		entry := getEntry(entry_id)
		
		sig_id_str := ctx.Params["sig_id"]
		sig_id, err := strconv.Atoi(sig_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to remove entry signature: error parsing signature id: ", err.Error())
			return
		}
		if sig_id >= len(entry.Signatures()){
			abortMessage = fmt.Sprint("Failed to remove entry signature: bad entry signature id: ", sig_id_str)
			break
		} 
		
		if entry.Unsign(sig_id) {
			ctx.Header().Add("Location", abortReturn)
			ctx.WriteHeader(303)
		} else {
			abortMessage = fmt.Sprint("Failed to remove entry signature #", sig_id)
		}
		
		storeEntry(entry_id)
		
	case "signEntry":
		entry_id_str := ctx.Params["id"]
		abortReturn = fmt.Sprint("/entries/", entry_id_str)
		entry_id, err := strconv.Atoi(entry_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to sign entry: error parsing entry id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(entry_id) {
			abortMessage = fmt.Sprint("Failed to sign entry: bad entry id: ", entry_id_str)
			return
		}
		entry := getEntry(entry_id)
		
		key_id_str := ctx.Params["key"]
		key_id, err := strconv.Atoi(key_id_str)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to sign entry: error parsing key id: ", err.Error())
			return
		}
		if !templateIsValidEntryId(entry_id) {
			abortMessage = fmt.Sprint("Failed to sign entry: bad key id: ", key_id_str)
			return
		}
		var key notaryapi.PrivateKey
		var ok bool
		_key := getKey(key_id)
		if key, ok = _key.(notaryapi.PrivateKey); !ok {
			abortMessage = fmt.Sprint("Failed to sign entry: key with id ", key_id_str, " is not a private key")
			return
		}
		
		err = entry.Sign(rand.Reader, key)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to sign entry: error while signing: ", err.Error())
			return
		}
		
		storeEntry(entry_id)
		
	case "genEntry":
		abortReturn = fmt.Sprint("/entries/add")
		
		sid := ctx.Params["datatype"]
		id, err := strconv.Atoi(sid)
		if err != nil {
			abortMessage = fmt.Sprint("Failed to generate entry: error data type: ", err.Error())
			return
		}
		
		entry := NewEntryOfType(EntryDataType(id))
		if entry == nil {
			abortMessage = fmt.Sprint("Failed to generate entry: unsupported data type: ", id)
			return
		}
		
		chainid := ctx.Params["chainid"]
		binaryChainID, err := notaryapi.DecodeBinary(&chainid) 
		if err != nil {
			abortMessage = fmt.Sprint("Invalid chain id: ", chainid)
			return
		} else{
			entry.ChainID = binaryChainID
		}
				
				
		
		addEntry(entry)
		
		ctx.Header().Add("Location", fmt.Sprintf("/entries/%d", Settings.NextEntryID-1))
		ctx.WriteHeader(303)
		
	default:
		abortReturn = fmt.Sprint("/entries")
		abortMessage = fmt.Sprint("Unknown action: ", action)
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
		
		bName := make ([][]byte, 0, 5)

		
			
		level0 := ctx.Params["level0"]
		if len(level0) > 0 {
			bName = append(bName, []byte(level0))
		}	
		level1 := ctx.Params["level1"]
		if len(level1) > 0 {
			bName = append(bName, []byte(level1))
		}		
		level2 := ctx.Params["level2"]
		if len(level2) > 0 {
			bName = append(bName, []byte(level2))
		}		
		level3 := ctx.Params["level3"]
		if len(level3) > 0 {
			bName = append(bName, []byte(level3))
		}		
		level4 := ctx.Params["level4"]		
		if len(level4) > 0 {
			bName = append(bName, []byte(level4))
		}		
		fmt.Println("level0:%v", level0)			
		fmt.Println("bName[0]%v", string(bName[0]))
		chain := new(notaryapi.Chain)
		chain.Name = bName	
		chain.GenerateIDFromName()			
		
		/*	
		server := fmt.Sprintf(`http://%s/v1`, Settings.Server)
		data := url.Values{}

		data.Set("datatype", "chain")
		data.Set("format", "binary")
		binaryChain,_ := chain.MarshalBinary()
		data.Set("chain", notaryapi.EncodeBinary(&binaryChain))
		
		resp, err := http.PostForm(server, data)	
		body, err := ioutil.ReadAll(resp.Body)
		
		resp.Body.Close()			
		*/
		

		err := factomapi.RevealChain(1, chain, nil)
				
		if err != nil {
			abortMessage = fmt.Sprint("An error occured while submitting the entry (entry may have been accepted by the server but was not locally flagged as such): ", err.Error())
			return
		}

		handleMessage(ctx, "Sumbitted", "The server has received your request to create a new chain with id: " + chain.ChainID.String())
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
	var externalHashes [][]byte
	
	defer func(){
		if action == "submit" {
			tmpl = "entrysubmit.gwp"
		} else {
			tmpl = "entry.gwp"
		}
		
		fentry, ok := entries[entry_id]
		
		if ok{
			newdata = string (fentry.Entry.Data())
			if fentry.Entry.ExtIDs != nil{
				externalHashes = fentry.Entry.ExtIDs
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
func handleFBlock(ctx *web.Context, hashStr string) {
	
	var title, error_str, fbBatchStatus string	
	hash,_ := notaryapi.HexToHash(hashStr)
	fBlock, _ := db.FetchFBlockByHash(hash)
	fbBatch, _ := db.FetchFBBatchByHash(hash)
	
	for _, fbEntry := range fBlock.FBEntries {
		fbHash, _ := db.FetchEBHashByMR(fbEntry.MerkleRoot)
		fbEntry.SetHash(fbHash.Bytes)
	}
	
	if fbBatch != nil {
		fbBatchStatus = "existing"
	}
	
	if fBlock == nil {
		handleMessage(ctx, "Not Found", "Factom Block not found for hash: " + hashStr)
		return
	}	
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "fblock.gwp",
			"fBlock": fBlock,	
			"fbHash": hashStr,	
			"fbBatch": fbBatch,	
			"fbBatchStatus": fbBatchStatus,
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
	fmt.Println("len(*eBlocks):%v", len(*eBlocks))
	sort.Sort(byEBlockID(*eBlocks))
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "chain.gwp",
			"eBlocks": *eBlocks,		
			"chain": chain,
		})
		if r != nil {
			handleError(ctx, r)
		}
	}()
	
}

func handleAllFBlocks(ctx *web.Context) {
	var title, error_str string	

	fBlocks, _ := db.FetchAllFBlocks()
	sort.Sort(byBlockID(fBlocks))
	
	defer func() {
		r := safeWrite(ctx, 200, map[string]interface{} {
			"Title": title,
			"Error": error_str,
			"ContentTmpl": "fblocks.gwp",
			"fBlocks": fBlocks,		
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
		handleFBlock(ctx, inputhash)
		
	case "extHash":
		handleMessage(ctx, "Not implemented yet", "Sorry this feature (External ID Search) is still under construction.")	
		
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

// array sorting implementation
type byBlockID []notaryapi.FBlock
func (f byBlockID) Len() int { 
  return len(f) 
} 
func (f byBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID > f[j].Header.BlockID
} 
func (f byBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 

// array sorting implementation
type byEBlockID []notaryapi.Block
func (f byEBlockID) Len() int { 
  return len(f) 
} 
func (f byEBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID > f[j].Header.BlockID
} 
func (f byEBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 
