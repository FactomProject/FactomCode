package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcutil"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gobundle"
	"github.com/firelizzard18/gocoding"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"os"
	"time"
	"log"
	"encoding/binary"
	"encoding/csv"
 
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/database/ldb"	
	"code.google.com/p/gcfg"	

)

var  (
	wclient *btcrpcclient.Client	//rpc client for btcwallet rpc server
	dclient *btcrpcclient.Client	//rpc client for btcd rpc server

 	currentAddr btcutil.Address
	tickers [2]*time.Ticker
	db database.Db // database
	chainIDMap map[string]*notaryapi.Chain // ChainIDMap with chainID string([32]byte) as key
	//chainNameMap map[string]*notaryapi.Chain // ChainNameMap with chain name string as key	
	fchain *notaryapi.FChain	//Factom Chain
	
//	fbBatches []*notaryapi.FBBatch
	fbBatches *FBBatches
	fbBatch *notaryapi.FBBatch
)

var (
 	logLevel = "DEBUG"
	portNumber int = 8083  	
	sendToBTCinSeconds = 60
	applicationName = "factom/restapi"
	dataStorePath = "/tmp/store/seed/"
	ldbpath = "/tmp/ldb9"
	//BTC:
//	addrStr = "movaFTARmsaTMk3j71MpX8HtMURpsKhdra"
	walletPassphrase = "lindasilva"
	certHomePath = "btcwallet"
	rpcClientHost = "localhost:18332"	//btcwallet rpcserver address
	rpcClientEndpoint = "ws"
	rpcClientUser = "testuser"
	rpcClientPass = "notarychain"
	btcTransFee float64 = 0.0001

	certHomePathBtcd = "btcd"
	rpcBtcdHost = "localhost:18334"		//btcd rpcserver address
	
)

type FBBatches struct {
	batches []*notaryapi.FBBatch
	batchMutex 	sync.Mutex
}

func loadConfigurations(){
	cfg := struct {
		App struct{
			PortNumber	int		
			ApplicationName string
			LdbPath	string
			DataStorePath string
	    }
		Btc struct{
			BTCPubAddr string
			SendToBTCinSeconds int		
			WalletPassphrase string	
			CertHomePath string
			RpcClientHost string
			RpcClientEndpoint string
			RpcClientUser string
			RpcClientPass string
			BtcTransFee float64
	    }		
		Log struct{
	    	LogLevel string
		}
    }{}
	
	wd, err := os.Getwd()
	if err != nil{
		log.Println(err)
	}	
	err = gcfg.ReadFileInto(&cfg, wd+"/restapi.conf")
	if err != nil{
		log.Println(err)
		log.Println("Server starting with default settings...")
	} else {
	
		//setting the variables by the valued form the config file
		logLevel = cfg.Log.LogLevel	
		applicationName = cfg.App.ApplicationName
		portNumber = cfg.App.PortNumber
		dataStorePath = cfg.App.DataStorePath
		ldbpath = cfg.App.LdbPath
//		addrStr = cfg.Btc.BTCPubAddr
		sendToBTCinSeconds = cfg.Btc.SendToBTCinSeconds 
		walletPassphrase = cfg.Btc.WalletPassphrase
		certHomePath = cfg.Btc.CertHomePath
		rpcClientHost = cfg.Btc.RpcClientHost
		rpcClientEndpoint = cfg.Btc.RpcClientEndpoint
		rpcClientUser = cfg.Btc.RpcClientUser
		rpcClientPass = cfg.Btc.RpcClientPass
		btcTransFee	  = cfg.Btc.BtcTransFee
	}
	
}

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}


func initWithBinary(chain *notaryapi.Chain) {
	matches, err := filepath.Glob(dataStorePath + chain.ChainID.String() + "/store.*.block") // need to get it from a property file??
	if err != nil {
		panic(err)
	}

	chain.Blocks = make([]*notaryapi.Block, len(matches))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(notaryapi.Block)
		err = block.UnmarshalBinary(data)
		if err != nil {
			panic(err)
		}
		block.Chain = chain
		block.IsSealed = true
		chain.Blocks[num] = block
		num++
	}
	
	//Create an empty block and append to the chain
	if len(chain.Blocks) == 0{
		chain.NextBlockID = 0		
		newblock, _ := notaryapi.CreateBlock(chain, nil, 10)
		
		chain.Blocks = append(chain.Blocks, newblock)	
	} else{
		chain.NextBlockID = uint64(len(chain.Blocks))			
		newblock,_ := notaryapi.CreateBlock(chain, chain.Blocks[len(chain.Blocks)-1], 10)
		chain.Blocks = append(chain.Blocks, newblock)
	}
	
	//Get the unprocessed entries in db for the past # of mins for the open block
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	if chain.Blocks[chain.NextBlockID].IsSealed == true {
		panic ("chain.Blocks[chain.NextBlockID].IsSealed for chain:" + chain.ChainID.String())
	}
	chain.Blocks[chain.NextBlockID].EBEntries, _ = db.FetchEBEntriesFromQueue(&chain.ChainID.Bytes, &binaryTimestamp)		
}

func initDB() {
	
	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	
	if err != nil{
		log.Println("err opening db: %v", err)

	}
	
	if db == nil{
		log.Println("Creating new db ...")			
		db, err = ldb.OpenLevelDB(ldbpath, true)

		if err!=nil{
			panic(err)
		} 		
	}
	log.Println("Database started from: " + ldbpath)	

}

func init() { 
	loadConfigurations()
	gobundle.Setup.Application.Name = applicationName
	gobundle.Init()
	
	initDB()
		
	initChains() 
	
	dynrsrc.Start(watchError, readError)
	notaryapi.StartDynamic(gobundle.DataFile("html.gwp"), readError)
	
	for _, chain := range chainIDMap {
		initWithBinary(chain)
			
		fmt.Println("Loaded", len(chain.Blocks)-1, "blocks for chain: " + chain.ChainID.String())
	
		for i := 0; i < len(chain.Blocks); i = i + 1 {
			if uint64(i) != chain.Blocks[i].Header.BlockID {
				panic(errors.New("BlockID does not equal index"))
			}
		}
	
	}

	// init FactomChain
	initFChain()
	fmt.Println("Loaded", len(fchain.Blocks)-1, "Factom blocks for chain: "+ notaryapi.EncodeBinary(fchain.ChainID))
	
	// init fbBatches, fbBatch
//	fbBatches = make([]*notaryapi.FBBatch, 0, 100)
	fbBatches = &FBBatches {
		batches: make([]*notaryapi.FBBatch, 0, 100),
	}
	fbBatch := &notaryapi.FBBatch {
		FBlocks: make([]*notaryapi.FBlock, 0, 10),
	}
	fbBatches.batches = append(fbBatches.batches, fbBatch)

	// create EBlocks and FBlock every 60 seconds
	tickers[0] = time.NewTicker(time.Second * time.Duration(sendToBTCinSeconds)) 

	// write 10 FBlock in a batch to BTC every 10 minutes
	tickers[1] = time.NewTicker(time.Minute * 2)	//10)

	go func() {
		for _ = range tickers[0].C {
			fmt.Println("in tickers[0]: newEntryBlock & newFactomBlock")
		
			for _, chain := range chainIDMap {
				eblock, blkhash := newEntryBlock(chain)
				if eblock != nil{
					fchain.AddFBEntry(eblock, blkhash)
				}
				save(chain)
			}
			
			fbBlock := newFactomBlock(fchain)
			if fbBlock != nil {
				// mark the start block of a FBBatch
				fmt.Println("in tickers[0]: len(fbBatch.FBlocks)=", len(fbBatch.FBlocks))
				if len(fbBatch.FBlocks) == 0 {
					fbBlock.Header.BatchFlag = byte(1)
				}
				fbBatch.FBlocks = append(fbBatch.FBlocks, fbBlock)
				fmt.Println("in tickers[0]: ADDED FBBLOCK: len(fbBatch.FBlocks)=", len(fbBatch.FBlocks))
			}
			
			saveFChain(fchain)									
		}
	}()

	
	go func() {
		for _ = range tickers[1].C {
			fmt.Println("in tickers[1]: new FBBatch. len(fbBatch.FBlocks)=", len(fbBatch.FBlocks))
			
			// skip empty fbBatch.
			if len(fbBatch.FBlocks) > 0 {
				doneBatch := fbBatch
				fbBatch = &notaryapi.FBBatch {
					FBlocks: make([]*notaryapi.FBlock, 0, 10),
				}
				
				fbBatches.batchMutex.Lock()
				fbBatches.batches = append(fbBatches.batches, doneBatch)
				fbBatches.batchMutex.Unlock()
			
				fmt.Printf("in tickers[1]: doneBatch=%#v\n", doneBatch)
			
				// go routine here?
				saveFBBatchMerkleRoottoBTC(doneBatch)
			}
		}
	}()
	
}


func main() {
	
	//addrStr := "muhXX7mXoMZUBvGLCgfjuoY2n2mziYETYC"
	//addrStr := "movaFTARmsaTMk3j71MpX8HtMURpsKhdra"
	
	err := initRPCClient()
	if err != nil {
		log.Fatalf("cannot init rpc client: %s", err)
	}
	defer shutdown()
	
	if err := initWallet(); err != nil {
		log.Fatalf("cannot init wallet: %s", err)
	}
	
	//doEntries()

	
	flag.Parse()
	defer func() {
		tickers[0].Stop()
		tickers[1].Stop()
		dynrsrc.Stop()
		db.Close()
	}()
	http.HandleFunc("/", serveRESTfulHTTP)
	err = http.ListenAndServe(":"+strconv.Itoa(portNumber), nil)
	if err != nil {
		panic(err)
	}

}


func fileNotExists(name string) (bool) {
  _, err := os.Stat(name)
  if os.IsNotExist(err) {
    return true
  }
  return err != nil
}


func save(chain *notaryapi.Chain) {
	if len(chain.Blocks)==0{
		log.Println("no blocks to save for chain: " + chain.ChainID.String())
		return
	}
	
	bcp := make([]*notaryapi.Block, len(chain.Blocks))

	chain.BlockMutex.Lock()
	copy(bcp, chain.Blocks)
	chain.BlockMutex.Unlock()

	
	for i, block := range bcp {
		//the open block is not saved
		if block.IsSealed == false {
			continue
		}
		data, err := block.MarshalBinary()
		if err != nil {
			panic(err)
		}

		strChainID := chain.ChainID.String()
		if fileNotExists (dataStorePath + strChainID){
			err:= os.MkdirAll(dataStorePath + strChainID, 0777)
			if err==nil{
				log.Println("Created directory " + dataStorePath + strChainID)
			} else{
				log.Println(err)
			}
		}

		err = ioutil.WriteFile(fmt.Sprintf(dataStorePath + strChainID + "/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func serveRESTfulHTTP(w http.ResponseWriter, r *http.Request) {
	var resource interface{}
	var err *notaryapi.Error
	var buf bytes.Buffer

	path, method, accept, form, err := parse(r)

	defer func() {
		switch accept {
		case "text":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		case "json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")

		case "xml":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")

		case "html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		if err != nil {
			var r *notaryapi.Error

			buf.Reset()
			r = notaryapi.Marshal(err, accept, &buf, false)
			if r != nil {
				err = r
			}
			w.WriteHeader(err.HTTPCode)
		}

		//buf.WriteTo(w)
		if resource != nil {
			//Send back entry hash
			w.Write(resource.([]byte)) 
		}else{
			w.Write([]byte("\n\n"))
		}
	}()

	switch method {
	case "GET":
		//resource, err = find(path)

	case "POST":
		if len(path) != 1 {
			err = notaryapi.CreateError(notaryapi.ErrorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}

		datatype := form.Get("datatype") 
		switch datatype {
			case "chain":
				resource, err = postChain("/"+strings.Join(path, "/"), form)
			default:
				resource, err = postEntry("/"+strings.Join(path, "/"), form)
		}

	default:
		err = notaryapi.CreateError(notaryapi.ErrorBadMethod, fmt.Sprintf(`The HTTP %s method is not supported`, method))
		return
	}

	if err != nil {
		resource = err
	}

	alt := false
	for _, s := range form["byref"] {
		b, err := strconv.ParseBool(s)
		if err == nil {
			alt = b
			break
		}
	}

	err = notaryapi.Marshal(resource, accept, &buf, alt)
}


func postEntry(context string, form url.Values) (interface{}, *notaryapi.Error) {
	newEntry := new(notaryapi.Entry)
	format, data := form.Get("format"), form.Get("entry")

	switch format {
	case "", "json":
		reader := gocoding.ReadString(data)
		err := notaryapi.UnmarshalJSON(reader, newEntry)
		if err != nil {
			return nil, notaryapi.CreateError(notaryapi.ErrorJSONUnmarshal, err.Error())
		}

	case "xml":
		err := xml.Unmarshal([]byte(data), newEntry)
		if err != nil {
			return nil, notaryapi.CreateError(notaryapi.ErrorXMLUnmarshal, err.Error())
		}
	case "binary":
		binaryEntry,_ := notaryapi.DecodeBinary(&data)
		fmt.Println("data:%v", data)
		err := newEntry.UnmarshalBinary(binaryEntry)
		if err != nil {
			return nil, notaryapi.CreateError(notaryapi.ErrorXMLUnmarshal, err.Error())
		}
	default:
		return nil, notaryapi.CreateError(notaryapi.ErrorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}

	return processNewEntry(newEntry)
}


func processNewEntry(newEntry *notaryapi.Entry) ([]byte, *notaryapi.Error) {
	if newEntry == nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Entity to be POSTed is nil`)
	}
	
	chain := chainIDMap[newEntry.ChainID.String()]
	if chain == nil{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `This chain is not supported`) //ErrorInternal?
	}
	
	// store the new entry in db
	entryBinary, _ := newEntry.MarshalBinary()
		fmt.Println("entryBinary:%v", notaryapi.EncodeBinary(&entryBinary))	
	hash, _ := notaryapi.CreateHash(newEntry)
	db.InsertEntryAndQueue( hash, &entryBinary, newEntry, &chain.ChainID.Bytes)

	chain.BlockMutex.Lock()
	err := chain.Blocks[len(chain.Blocks)-1].AddEBEntry(newEntry)
	chain.BlockMutex.Unlock()

	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}

	return hash.Bytes, nil
}

func postChain(context string, form url.Values) (interface{}, *notaryapi.Error) {
	newChain := new(notaryapi.Chain)
	format, data := form.Get("format"), form.Get("chain")

	switch format {
	case "binary":
		binaryChain,_ := notaryapi.DecodeBinary(&data)
		err := newChain.UnmarshalBinary(binaryChain)	
		newChain.GenerateIDFromName()		
		if err != nil {
			return nil, notaryapi.CreateError(notaryapi.ErrorInternal, err.Error())
		}
	default:
		return nil, notaryapi.CreateError(notaryapi.ErrorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}

	if newChain == nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Chain is nil`)
	}
	
	chain := chainIDMap[newChain.ChainID.String()]
	if chain != nil{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `This chain is already existing`) //ErrorInternal?
	}
	
	// Store the new chain in db
	db.InsertChain(newChain)
	
	// Chain initialization
	initWithBinary(newChain)	
	fmt.Println("Loaded", len(newChain.Blocks)-1, "blocks for chain: " + newChain.ChainID.String())	
	
	// Add the new chain in the chainIDMap
	chainIDMap[newChain.ChainID.String()] = newChain	

	ExportDataFromDbToFile()
	
	return newChain.ChainID.Bytes, nil
}

/*
func createNewChain(chainName string) (chain *notaryapi.Chain){
		chain = new (notaryapi.Chain)
		chain.Name = chainName
		chain.GenerateIDFromName(chainName)
		
		db.InsertChain(chain)
		chainIDMap[chain.ChainID.String()] = chain
		
		return chain
}
*/

func saveFChain(chain *notaryapi.FChain) {
	if len(chain.Blocks)==0{
		//log.Println("no blocks to save for chain: " + string (*chain.ChainID))
		return
	}
	
	bcp := make([]*notaryapi.FBlock, len(chain.Blocks))

	chain.BlockMutex.Lock()
	copy(bcp, chain.Blocks)
	chain.BlockMutex.Unlock()

	for i, block := range bcp {
		//the open block is not saved
		if block.IsSealed == false {
			continue
		}
		
		data, err := block.MarshalBinary()
		if err != nil {
			panic(err)
		}

		strChainID := notaryapi.EncodeBinary(chain.ChainID)
		if fileNotExists (dataStorePath + strChainID){
			err:= os.MkdirAll(dataStorePath + strChainID, 0777)
			if err==nil{
				log.Println("Created directory " + dataStorePath + strChainID)
			} else{
				log.Println(err)
			}
		}
		err = ioutil.WriteFile(fmt.Sprintf(dataStorePath + strChainID + "/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func initFChain() {
	fchain = new (notaryapi.FChain)

	barray := (make([]byte, 32))
	fchain.ChainID = &barray
	
	matches, err := filepath.Glob(dataStorePath + notaryapi.EncodeBinary(fchain.ChainID) + "/store.*.block") // need to get it from a property file??
	if err != nil {
		panic(err)
	}

	fchain.Blocks = make([]*notaryapi.FBlock, len(matches))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(notaryapi.FBlock)
		err = block.UnmarshalBinary(data)
		if err != nil {
			panic(err)
		}
		block.Chain = fchain
		block.IsSealed = true
		
		fchain.Blocks[num] = block
		num++
	}
	//Create an empty block and append to the chain
	if len(fchain.Blocks) == 0{
		fchain.NextBlockID = 0
		newblock, _ := notaryapi.CreateFBlock(fchain, nil, 10)
		fchain.Blocks = append(fchain.Blocks, newblock)
		
	} else{
		fchain.NextBlockID = uint64(len(fchain.Blocks))			
		newblock,_ := notaryapi.CreateFBlock(fchain, fchain.Blocks[len(fchain.Blocks)-1], 10)	
		fchain.Blocks = append(fchain.Blocks, newblock)		
	}	
	
	//Get the unprocessed entries in db for the past # of mins for the open block
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	if fchain.Blocks[fchain.NextBlockID].IsSealed == true {
		panic ("fchain.Blocks[fchain.NextBlockID].IsSealed for chain:" + notaryapi.EncodeBinary(fchain.ChainID))
	}
	fchain.Blocks[fchain.NextBlockID].FBEntries, _ = db.FetchFBEntriesFromQueue(&binaryTimestamp)			

}

func ExportDbToFile(fbHash *notaryapi.Hash) {
	
	if fileNotExists( dataStorePath+"csv/") {
		os.MkdirAll(dataStorePath+"csv/" , 0755)
	}
	
	//write the records to a csv file: 
	file, err := os.Create(dataStorePath+"csv/" + fbHash.String() + ".csv")
	if err != nil {panic(err)}
    defer file.Close()
    writer := csv.NewWriter(file)
 
 	ldbMap, err := db.FetchAllDBRecordsByFBHash(fbHash)   
	
 	if err != nil{
 		log.Println(err)
 		return
 	}
 	
    for key, value := range ldbMap{
	    //csv header: key, value
	    writer.Write([]string {key, value})
    }
	writer.Flush()	    
}

func ExportDataFromDbToFile() {
	
	if fileNotExists( dataStorePath+"csv/") {
		os.MkdirAll(dataStorePath+"csv/" , 0755)
	}
	
	//write the records to a csv file: 
	file, err := os.Create(dataStorePath+"csv/"  + "supportdata.csv")
	if err != nil {panic(err)}
    defer file.Close()
    writer := csv.NewWriter(file)
 
 	ldbMap, err := db.FetchSupportDBRecords()   
	
 	if err != nil{
 		log.Println(err)
 		return
 	}
 	
    for key, value := range ldbMap{
	    //csv header: key, value
	    writer.Write([]string {key, value})
    }
	writer.Flush()	    
}

func initChains() {
//	initChainIDs()
	
	chainIDMap = make(map[string]*notaryapi.Chain)
	//chainNameMap = make(map[string]*notaryapi.Chain)

	chains, err := db.FetchAllChainsByName(nil)
	
	if err != nil{
		panic (err)
	}
	
	for _, chain := range *chains {
		var newChain = chain
		chainIDMap[newChain.ChainID.String()] = &newChain	
		//chainIDMap[string(chain.ChainID.Bytes)] = &chain			
	}
}

//for testing - to be moved into db-------------------------------------

var chainIDs [][]byte

func initChainIDs() {
//	chainMap = make(map[string]*notaryapi.Chain)

	barray1 := make([]byte, 32)
	barray1[0] = byte(1)
	chainIDs = [][]byte{barray1}
	chain1 := new (notaryapi.Chain)
	chain1.ChainID = new (notaryapi.Hash)
	chain1.ChainID.Bytes = chainIDs[0]
	chain1.Name = [][]byte {[]byte("apple"), []byte("phone")}
	db.InsertChain(chain1)
//	chainMap[string(chainIDs[0])] = chain1
	
	barray2 := make([]byte, 32)
	barray2[0] = byte(2)
	chainIDs = append(chainIDs, barray2)
	chain2 := new (notaryapi.Chain)
	chain2.ChainID = new (notaryapi.Hash)
	chain2.ChainID.Bytes = chainIDs[1]
	chain2.Name = [][]byte {[]byte("apple"), []byte("mac")}	
	db.InsertChain(chain2)	
//	chainMap[string(chainIDs[1])] = chain2	

}


