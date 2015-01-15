package main

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"errors"
	//"flag"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcutil"
	"github.com/FactomProject/dynrsrc"
	"github.com/FactomProject/gobundle"
	"github.com/FactomProject/gocoding"
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
	"reflect"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/wallet"

)  
 
var  (
	wclient *btcrpcclient.Client	//rpc client for btcwallet rpc server
	dclient *btcrpcclient.Client	//rpc client for btcd rpc server

 	currentAddr btcutil.Address
	tickers [2]*time.Ticker
	db database.Db // database
	chainIDMap map[string]*notaryapi.EChain // ChainIDMap with chainID string([32]byte) as key
	//chainNameMap map[string]*notaryapi.Chain // ChainNameMap with chain name string as key	
	dchain *notaryapi.DChain	//Directory Block Chain
	cchain *notaryapi.CChain	//Entry Credit Chain
	
	creditsPerChain int32 = 10
	creditsPerFactoid uint64 = 1000
	eCreditMap map[string]int32 // eCreditMap with public key string([32]byte) as key, credit balance as value
	prePaidEntryMap map[string]int32 // Paid but unrevealed entries string(Etnry Hash) as key, Number of payments as value		
	
//	dbBatches []*notaryapi.FBBatch
	dbBatches *DBBatches
	dbBatch *notaryapi.DBBatch
	
	//Map to store export csv files
	serverDataFileMap map[string]string
)

var (
 	logLevel = "DEBUG"
	portNumber int = 8083  	
	sendToBTCinSeconds = 600
	directoryBlockInSeconds = 60
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

type DBBatches struct {
	batches []*notaryapi.DBBatch
	batchMutex 	sync.Mutex
}

func loadConfigurations(){
	cfg := struct {
		App struct{
			PortNumber	int		
			ApplicationName string
			LdbPath	string
			DataStorePath string
			DirectoryBlockInSeconds int				
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
		directoryBlockInSeconds = cfg.App.DirectoryBlockInSeconds
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


func initWithBinary(chain *notaryapi.EChain) {
	matches, err := filepath.Glob(dataStorePath + chain.ChainID.String() + "/store.*.block") // need to get it from a property file??
	if err != nil {
		panic(err)
	}

	chain.Blocks = make([]*notaryapi.EBlock, len(matches))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(notaryapi.EBlock)
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

	// init Directory Block Chain
	initDChain()
	fmt.Println("Loaded", len(dchain.Blocks)-1, "Directory blocks for chain: "+ dchain.ChainID.String())
	
	
	// init Entry Credit Chain
	initCChain()
	fmt.Println("Loaded", len(cchain.Blocks)-1, "Entry Credit blocks for chain: "+ cchain.ChainID.String())	
	
	// init dbBatches, dbBatch
	dbBatches = &DBBatches {
		batches: make([]*notaryapi.DBBatch, 0, 100),
	}
	dbBatch := &notaryapi.DBBatch {
		DBlocks: make([]*notaryapi.DBlock, 0, 10),
	}
	dbBatches.batches = append(dbBatches.batches, dbBatch)
	
	// init the export file list for client distribution
	initServerDataFileMap()
		

	// create EBlocks and FBlock every 60 seconds
	tickers[0] = time.NewTicker(time.Second * time.Duration(directoryBlockInSeconds)) 

	// write 10 FBlock in a batch to BTC every 10 minutes
	tickers[1] = time.NewTicker(time.Second * time.Duration(sendToBTCinSeconds))	

	go func() {
		for _ = range tickers[0].C {
			fmt.Println("in tickers[0]: newEntryBlock & newFactomBlock")
		
			// Entry Chains
			for _, chain := range chainIDMap {
				eblock := newEntryBlock(chain)
				if eblock != nil{
					dchain.AddDBEntry(eblock)
				}
				save(chain)
			}
	
			// Entry Credit Chain
			cBlock := newEntryCreditBlock(cchain)
			if cBlock != nil{
				dchain.AddCBlockToDBEntry(cBlock)
			}			
			saveCChain(cchain)		
			
			// Directory Block chain
			dbBlock := newDirectoryBlock(dchain)
			if dbBlock != nil {
				// mark the start block of a DBBatch
				fmt.Println("in tickers[0]: len(dbBatch.DBlocks)=", len(dbBatch.DBlocks))
				if len(dbBatch.DBlocks) == 0 {
					dbBlock.Header.BatchFlag = byte(1)
				}
				dbBatch.DBlocks = append(dbBatch.DBlocks, dbBlock)
				fmt.Println("in tickers[0]: ADDED FBBLOCK: len(dbBatch.DBlocks)=", len(dbBatch.DBlocks))
			}
			saveDChain(dchain)				
								
		}
	}()

	
	go func() {
		for _ = range tickers[1].C {
			fmt.Println("in tickers[1]: new FBBatch. len(dbBatch.DBlocks)=", len(dbBatch.DBlocks))
			
			// skip empty dbBatch.
			if len(dbBatch.DBlocks) > 0 {
				doneBatch := dbBatch
				dbBatch = &notaryapi.DBBatch {
					DBlocks: make([]*notaryapi.DBlock, 0, 10),
				}
				
				dbBatches.batchMutex.Lock()
				dbBatches.batches = append(dbBatches.batches, doneBatch)
				dbBatches.batchMutex.Unlock()
			
				fmt.Printf("in tickers[1]: doneBatch=%#v\n", doneBatch)
			
				// go routine here?
				saveDBBatchMerkleRoottoBTC(doneBatch)
			}
		}
	}()
	
}


func main() {
/**/

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

	
	//flag.Parse()
	defer func() {
		tickers[0].Stop()
		tickers[1].Stop()
		dynrsrc.Stop()
		db.Close()
	}()
/*	
	err :=http.ListenAndServe(":8081", http.FileServer(http.Dir("/tmp/store/seed/csv")))
	if err != nil {
		panic(err)
	}
*/		
	http.HandleFunc("/", serveRESTfulHTTP)
	err1 := http.ListenAndServe(":"+strconv.Itoa(portNumber), nil)
	if err1 != nil {
		panic(err1)
	}
	

}


func fileNotExists(name string) (bool) {
  _, err := os.Stat(name)
  if os.IsNotExist(err) {
    return true
  }
  return err != nil
}


func save(chain *notaryapi.EChain) {
	if len(chain.Blocks)==0{
		log.Println("no blocks to save for chain: " + chain.ChainID.String())
		return
	}
	
	bcp := make([]*notaryapi.EBlock, len(chain.Blocks))

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
	switch method {
	case "GET":
		//resource, err = getServerDataFileMap()

	case "POST":
		if len(path) != 1 {
			err = notaryapi.CreateError(notaryapi.ErrorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}

		datatype := form.Get("datatype")
		switch datatype {
		case "commitchain":
			var err error
			pub := new(notaryapi.Hash)
			//chainhash := new(notaryapi.Hash)
			chainID := new(notaryapi.Hash)			
			entrychainhash := new(notaryapi.Hash)
			entryhash := new(notaryapi.Hash)

			data, err := hex.DecodeString(r.Form.Get("data"))
			if err != nil { fmt.Println("hex:data", err) }

			sig, err := hex.DecodeString(r.Form.Get("signature"))
 			if err != nil { fmt.Println("hex:signature", err) }

			pub.Bytes, err = hex.DecodeString(r.Form.Get("pubkey"))
 			if err != nil { fmt.Println("hex:pubkey", err) }

 			if ( !wallet.VerifySlice(pub.Bytes, data, sig) ) {
				err = notaryapi.CreateError(notaryapi.ErrorVerifySignature, "commitchain Verify failed")
				break;
 			}
			
			data = data[8:]
			chainID.Bytes = data[:32]
			data = data[32:]

			entryhash.Bytes = data[:32]
			data = data[32:]
						
			entrychainhash.Bytes = data[:32]
			data = data[32:]

			buf := bytes.NewBuffer(data)
			var credits int32
    		err = binary.Read(buf, binary.BigEndian, &credits)
			if err != nil {
				fmt.Println("binary credits err:", err.Error())
			} else { fmt.Println("credits %v",credits)}


			resource, err = processCommitChain(entryhash, chainID, entrychainhash, pub, credits)

			if err != nil {
				fmt.Println("Error:", err.Error())
			}

		case "revealchain":
			c := new(notaryapi.EChain)
			bin, err := hex.DecodeString(r.Form.Get("data"))
			if err != nil {
				fmt.Println("hex:", err.Error())
			}
			c.UnmarshalBinary(bin)
			
			nerr := new(notaryapi.Error)
			resource, nerr = processRevealChain(c)

			if nerr != nil {
				fmt.Println("Error:", nerr.Error())
			}

		case "commitentry":
			var err error
			var credits int32				
			pub := new(notaryapi.Hash)
			hash := new(notaryapi.Hash)
			data, err := hex.DecodeString(r.Form.Get("data"))
			if err != nil { fmt.Println("hex:data", err) }

			sig, err := hex.DecodeString(r.Form.Get("signature"))
 			if err != nil { fmt.Println("hex:signature", err.Error()) }

			pub.Bytes, err = hex.DecodeString(r.Form.Get("pubkey"))
 			if err != nil { fmt.Println("hex:pubkey", err.Error()) }

 			if ( !wallet.VerifySlice(pub.Bytes, data, sig) ) {
				err = notaryapi.CreateError(notaryapi.ErrorVerifySignature, "commitentry Verify failed")
				break;
 			}

			timestamp := binary.BigEndian.Uint64(data[:8])
			data = data[8:]
			hash.Bytes = data[:32]
			data = data[32:]
			buf := bytes.NewBuffer(data[:4])
			err = binary.Read(buf, binary.BigEndian, &credits)
			if err != nil {
				fmt.Println("Error:", err.Error())
			}	
			resource, err = processCommitEntry(hash, pub, int64(timestamp), credits) 
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
		case "revealentry":
			e := new(notaryapi.Entry)
			bin, err := hex.DecodeString(r.Form.Get("entry"))
			if err != nil {
				fmt.Println("hex:", err.Error())
			}
			e.UnmarshalBinary(bin)
			nerr := new(notaryapi.Error)			
			resource, nerr = processRevealEntry(e)
			if err  != nil {
				fmt.Println("Error:", nerr.Error())
			}
			
		case "commitentry2":
			var err error
			pub := new(notaryapi.Hash)
			hash := new(notaryapi.Hash)
			data, err := hex.DecodeString(r.Form.Get("data"))
			if err != nil { fmt.Println("hex:data", err) }

			sig, err := hex.DecodeString(r.Form.Get("signature"))
 			if err != nil { fmt.Println("hex:signature", err.Error()) }

			pub.Bytes, err = hex.DecodeString(r.Form.Get("pubkey"))
 			if err != nil { fmt.Println("hex:pubkey", err.Error()) }

 			if ( !wallet.VerifySlice(pub.Bytes, data, sig) ) {
				err = notaryapi.CreateError(notaryapi.ErrorVerifySignature, "commitentry Verify failed")
				break;
 			}

			timestamp := binary.BigEndian.Uint64(data[:8])
			hash.Bytes = data[8:]
			resource, err = processCommitEntry(hash, pub, int64(timestamp), 1) // needs the credits from client ??
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
		case "revealentry2":
			e := new(notaryapi.Entry)
			bin, err := hex.DecodeString(r.Form.Get("entry"))
			if err != nil {
				fmt.Println("hex:", err.Error())
			}
			e.UnmarshalBinary(bin)
			resource, err = processRevealEntry(e)
			if err != nil {
				fmt.Println("Error:", err.Error())
			}			
		// (commit|reveal)chain
		case "chain":
			resource, err = postChain("/"+strings.Join(path, "/"), form)
		case "buycredit":
			pubKey, err := notaryapi.HexToHash(form.Get("ECPubKey")) 
			if err!=nil{
				fmt.Println("Error in parsing pubKey:", err.Error())
			}
			value, err := strconv.ParseUint(form.Get("factoidbase"), 10, 64)
			if err!=nil{
				fmt.Println("Error in parsing value:", err.Error())
			}	
			credits := value * creditsPerFactoid / 1000000000
			resource, err = processBuyEntryCredit(pubKey, int32(credits), pubKey)		
			printCreditMap()				
		case "getbalance":
			pubKey, err := notaryapi.HexToHash(form.Get("ECPubKey")) 
			if err!=nil{
				fmt.Println("Error in parsing pubKey:", err.Error())
			}
			resource, err = getEntryCreditBalance(pubKey)			
		case "filelist":
			resource, err = getServerDataFileMapJSON()			
		case "file":
			fileKey := form.Get("filekey")
			filename := serverDataFileMap[fileKey]
			http.ServeFile(w, r, dataStorePath + "csv/"+filename)
			return							
		default:
			err = notaryapi.CreateError(notaryapi.ErrorBadMethod, fmt.Sprintf(`The %s datatype is not supported`, datatype))
			return
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

	return processRevealEntry(newEntry)
}


func processRevealEntry(newEntry *notaryapi.Entry) ([]byte, *notaryapi.Error) {

	if newEntry == nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Entity to be POSTed is nil`)
	}


	chain := chainIDMap[newEntry.ChainID.String()]
	if chain == nil{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `This chain is not supported`) //ErrorInternal?
	}

	// store the new entry in db
	entryBinary, _ := newEntry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	db.InsertEntryAndQueue( entryHash, &entryBinary, newEntry, &chain.ChainID.Bytes)
	
	// Calculate the required credits
	credits := int32(binary.Size(entryBinary)/1000 + 1) 

	// Precalculate the key for prePaidEntryMap
	key := entryHash.String()
	chain.BlockMutex.Lock()
	// Delete the entry in the prePaidEntryMap in memory
	prepayment, ok := prePaidEntryMap[key]
	if ok && prepayment >= credits {
		delete (prePaidEntryMap, key)	// Only revealed once for multiple prepayments??
	} else{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Credit needs to paid first before reveal an entry:` + entryHash.String())
	}	
	err := chain.Blocks[len(chain.Blocks)-1].AddEBEntry(newEntry)
	chain.BlockMutex.Unlock()

	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}

	return entryHash.Bytes, nil
}

func processCommitEntry(entryHash *notaryapi.Hash, pubKey *notaryapi.Hash, 	timeStamp int64, credits int32) ([]byte, error) {
	// Make sure credits is negative
	if credits > 0 {
		credits = 0 - credits
	}	
	// Create PayEntryCBEntry
	cbEntry := notaryapi.NewPayEntryCBEntry(pubKey, entryHash, credits, timeStamp)
	
	cchain.BlockMutex.Lock()
	// Update the credit balance in memory
	creditBalance, _ := eCreditMap[pubKey.String()]
	if creditBalance + credits < 0 {
		return nil, errors.New("Not enough credit for public key:" + pubKey.String() + " Balance:" + fmt.Sprint(credits))
	}
	eCreditMap[pubKey.String()] = creditBalance + credits
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)
	// Update the prePaidEntryMapin memory
	payments, _ := prePaidEntryMap[entryHash.String()]	
	prePaidEntryMap[entryHash.String()]	= payments - credits
	cchain.BlockMutex.Unlock()	 

	return entryHash.Bytes, err
}

func processCommitChain(entryHash *notaryapi.Hash, chainIDHash *notaryapi.Hash, entryChainIDHash *notaryapi.Hash, pubKey *notaryapi.Hash, credits int32) ([]byte, error) {
	
	// Make sure credits is negative
	if credits > 0 {
		credits = 0 - credits
	}

	// Check if the chain id already exists
	_, existing := chainIDMap[chainIDHash.String()]
	if !existing {
		if chainIDHash.IsSameAs(dchain.ChainID) || chainIDHash.IsSameAs(cchain.ChainID) {
			existing = true
		}	
	}
	if existing {
		return nil, errors.New("Already existing chain id:" + chainIDHash.String())
	}
	
	// Precalculate the key and value pair for prePaidEntryMap
	key := getPrePaidChainKey(entryHash, chainIDHash)
		
	// Create PayChainCBEntry
	cbEntry := notaryapi.NewPayChainCBEntry(pubKey, entryHash, credits, chainIDHash, entryChainIDHash)
	
	cchain.BlockMutex.Lock()
	// Update the credit balance in memory
	creditBalance, _ := eCreditMap[pubKey.String()]
	if creditBalance + credits < 0 {
		return nil, errors.New("Insufficient credits for public key:" + pubKey.String() + " Balance:" + fmt.Sprint(credits))
	}	
	eCreditMap[pubKey.String()] = creditBalance + credits
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)
	// Update the prePaidEntryMap in memory
	payments, _ := prePaidEntryMap[key]	
	prePaidEntryMap[key] = payments - credits
	cchain.BlockMutex.Unlock()	 
 
	return chainIDHash.Bytes, err
}
func processBuyEntryCredit(pubKey *notaryapi.Hash, credits int32, factoidTxHash *notaryapi.Hash) ([]byte, error) {
	
	cbEntry := notaryapi.NewBuyCBEntry(pubKey, factoidTxHash, credits)
	cchain.BlockMutex.Lock()
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)
	// Update the credit balance in memory
	balance, _ := eCreditMap[pubKey.String()]
	eCreditMap[pubKey.String()] = balance + credits	
	cchain.BlockMutex.Unlock()	 

 
	return pubKey.Bytes, err
}

func processRevealChain(newChain *notaryapi.EChain) ([]byte, *notaryapi.Error) {
	// Check if the chain id already exists
	_, existing := chainIDMap[newChain.ChainID.String()]
	if !existing {
		if newChain.ChainID.IsSameAs(dchain.ChainID) || newChain.ChainID.IsSameAs(cchain.ChainID) {
			existing = true
		}	
	}
	if existing{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `This chain is already existing`) //ErrorInternal?
	}	
	
	if newChain.FirstEntry == nil{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `The first entry is required to create a new chain.`) //ErrorInternal?
	}	
	// Calculate the required credits
	binaryChain, _ := newChain.MarshalBinary()
	credits := int32(binary.Size(binaryChain)/1000 + 1) + creditsPerChain 
	
	// Remove the entry for prePaidEntryMap
	binaryEntry, _ := newChain.FirstEntry.MarshalBinary()
	firstEntryHash := notaryapi.Sha(binaryEntry)
	key := getPrePaidChainKey(firstEntryHash, newChain.ChainID)	
	prepayment, ok := prePaidEntryMap[key]
	if ok && prepayment >= credits {
		delete(prePaidEntryMap, key)
	} else{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Enough credits need to paid first before creating a new chain:` + newChain.ChainID.String())
	}
	// Store the new chain in db
	db.InsertChain(newChain)
		
	// Chain initialization
	initWithBinary(newChain)	
	fmt.Println("Loaded", len(newChain.Blocks)-1, "blocks for chain: " + newChain.ChainID.String())	
	
	// Add the new chain in the chainIDMap
	chainIDMap[newChain.ChainID.String()] = newChain	
	
	// store the new entry in db
	entryBinary, _ := newChain.FirstEntry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	db.InsertEntryAndQueue( entryHash, &entryBinary, newChain.FirstEntry, &newChain.ChainID.Bytes)

	newChain.BlockMutex.Lock()	
	err := newChain.Blocks[len(newChain.Blocks)-1].AddEBEntry(newChain.FirstEntry)
	newChain.BlockMutex.Unlock()	

	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, fmt.Sprintf(`Error while adding the First Entry to Block: %s`, err.Error()))
	}

	ExportDataFromDbToFile()
	
	return newChain.ChainID.Bytes, nil	
}

func getEntryCreditBalance(pubKey *notaryapi.Hash) ([]byte, error) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, eCreditMap[pubKey.String()])		 
	return buf.Bytes(), nil
}

func postChain(context string, form url.Values) (interface{}, *notaryapi.Error) {
	newChain := new(notaryapi.EChain)
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
		
	return processRevealChain(newChain) 
}

func saveDChain(chain *notaryapi.DChain) {
	if len(chain.Blocks)==0{
		//log.Println("no blocks to save for chain: " + string (*chain.ChainID))
		return
	}
	
	bcp := make([]*notaryapi.DBlock, len(chain.Blocks))

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

func saveCChain(chain *notaryapi.CChain) {
	if len(chain.Blocks)==0{
		//log.Println("no blocks to save for chain: " + string (*chain.ChainID))
		return
	}
	
	bcp := make([]*notaryapi.CBlock, len(chain.Blocks))

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

func initDChain() {
	dchain = new (notaryapi.DChain)

	//to be improved??
	dchain.ChainID = new (notaryapi.Hash)	
	barray := (make([]byte, 32))
	dchain.ChainID.SetBytes(barray)	
	
	
	matches, err := filepath.Glob(dataStorePath + dchain.ChainID.String() + "/store.*.block") // need to get it from a property file??
	if err != nil {
		panic(err)
	}

	dchain.Blocks = make([]*notaryapi.DBlock, len(matches))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(notaryapi.DBlock)
		err = block.UnmarshalBinary(data)
		if err != nil {
			panic(err)
		}
		block.Chain = dchain
		block.IsSealed = true
		
		dchain.Blocks[num] = block
		num++
	}
	//Create an empty block and append to the chain
	if len(dchain.Blocks) == 0{
		dchain.NextBlockID = 0
		newblock, _ := notaryapi.CreateDBlock(dchain, nil, 10)
		dchain.Blocks = append(dchain.Blocks, newblock)
		
	} else{
		dchain.NextBlockID = uint64(len(dchain.Blocks))			
		newblock,_ := notaryapi.CreateDBlock(dchain, dchain.Blocks[len(dchain.Blocks)-1], 10)	
		dchain.Blocks = append(dchain.Blocks, newblock)		
	}	
	
	//Get the unprocessed entries in db for the past # of mins for the open block
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	if dchain.Blocks[dchain.NextBlockID].IsSealed == true {
		panic ("dchain.Blocks[dchain.NextBlockID].IsSealed for chain:" + dchain.ChainID.String())
	}
	dchain.Blocks[dchain.NextBlockID].DBEntries, _ = db.FetchDBEntriesFromQueue(&binaryTimestamp)			

}

func initCChain() {
	
	eCreditMap = make(map[string]int32)
	prePaidEntryMap = make(map[string]int32)
	
	cchain = new (notaryapi.CChain)

	//to be improved??
	barray := (make([]byte, 32))
	barray[0] = 1
	cchain.ChainID = new (notaryapi.Hash)
	cchain.ChainID.SetBytes(barray)

	
	matches, err := filepath.Glob(dataStorePath + cchain.ChainID.String() + "/store.*.block") // need to get it from a property file??
	if err != nil {
		panic(err)
	}

	cchain.Blocks = make([]*notaryapi.CBlock, len(matches))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(notaryapi.CBlock)
		err = block.UnmarshalBinary(data)
		if err != nil {
			panic(err)
		}
		block.Chain = cchain
		block.IsSealed = true
		
		// Calculate the EC balance for each account 
		initializeECreditMap(block)
		
		cchain.Blocks[num] = block
		num++
	}
	//Create an empty block and append to the chain
	if len(cchain.Blocks) == 0{
		cchain.NextBlockID = 0
		newblock, _ := notaryapi.CreateCBlock(cchain, nil, 10)
		cchain.Blocks = append(cchain.Blocks, newblock)
		
	} else{
		cchain.NextBlockID = uint64(len(cchain.Blocks))			
		newblock,_ := notaryapi.CreateCBlock(cchain, cchain.Blocks[len(cchain.Blocks)-1], 10)	
		cchain.Blocks = append(cchain.Blocks, newblock)		
	}	
	
	//Get the unprocessed entries in db for the past # of mins for the open block
/*	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	if cchain.Blocks[cchain.NextBlockID].IsSealed == true {
		panic ("dchain.Blocks[dchain.NextBlockID].IsSealed for chain:" + notaryapi.EncodeBinary(dchain.ChainID))
	}
	dchain.Blocks[dchain.NextBlockID].DBEntries, _ = db.FetchDBEntriesFromQueue(&binaryTimestamp)			
*/
}


func ExportDbToFile(dbHash *notaryapi.Hash) {
	
	if fileNotExists( dataStorePath+"csv/") {
		os.MkdirAll(dataStorePath+"csv/" , 0755)
	}
	
	//write the records to a csv file: 
	filename := fmt.Sprintf("%v." + dbHash.String() + ".csv", time.Now().Unix())
	file, err := os.Create(dataStorePath+"csv/" + filename)
	
	if err != nil {panic(err)}
    defer file.Close()
    writer := csv.NewWriter(file)
 
 	ldbMap, err := db.FetchAllDBRecordsByDBHash(dbHash, cchain.ChainID)   
	
 	if err != nil{
 		log.Println(err)
 		return 
 	}
 	
    for key, value := range ldbMap{
	    //csv header: key, value
	    writer.Write([]string {key, value})
    }
	writer.Flush()	    
	
	// Add the file to the distribution list
	hash := notaryapi.Sha([]byte(filename))
	serverDataFileMap[hash.String()] = filename	
}

func ExportDataFromDbToFile() {
	
	if fileNotExists( dataStorePath+"csv/") {
		os.MkdirAll(dataStorePath+"csv/" , 0755)
	}
	
	//write the records to a csv file: 
	filename := fmt.Sprintf("%v.supportdata.csv",time.Now().Unix())
	file, err := os.Create(dataStorePath+"csv/"  + filename)
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
	
	// Add the file to the distribution list
	hash := notaryapi.Sha([]byte(filename))
	serverDataFileMap[hash.String()] = filename
	
}

func initChains() {
	//initChainIDs()
	
	
	chainIDMap = make(map[string]*notaryapi.EChain)
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


func initializeECreditMap(block *notaryapi.CBlock) {
	for _, cbEntry := range block.CBEntries{
		credits, _ := eCreditMap[cbEntry.PublicKey().String()]
		eCreditMap[cbEntry.PublicKey().String()] = credits + cbEntry.Credits()	
	}
}

func getPrePaidChainKey(entryHash *notaryapi.Hash, chainIDHash *notaryapi.Hash) string {
	return chainIDHash.String() + entryHash.String()
}

func printCreditMap(){
	
	fmt.Println("eCreditMap:")
	for key := range eCreditMap {
		fmt.Println("Key:", key, "Value", eCreditMap[key])
	}
}

func printPaidEntryMap(){
	
	fmt.Println("prePaidEntryMap:")
	for key := range prePaidEntryMap {
		fmt.Println("Key:", key, "Value", prePaidEntryMap[key])
	}
}

func printCChain(){
	
	fmt.Println("cchain:", cchain.ChainID.String())
	
	for i, block := range cchain.Blocks {
		if !block.IsSealed{
			continue
		}
		var buf bytes.Buffer
		err := factomapi.SafeMarshal(&buf, block.Header)
		
		fmt.Println("block.Header", string(i), ":", string(buf.Bytes()))	
		
		for _, cbentry := range block.CBEntries {
			t := reflect.TypeOf(cbentry)
			fmt.Println("cbEntry Type:", t.Name(), t.String())
			if strings.Contains(t.String(), "PayChainCBEntry"){
				fmt.Println("PayChainCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err!=nil{
					fmt.Println("Error:%v", err)
				}
				
				fmt.Println("PayChainCBEntry JSON",  ":", string(buf.Bytes()))			
						
			} else 	if strings.Contains(t.String(), "PayEntryCBEntry"){
				fmt.Println("PayEntryCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err!=nil{
					fmt.Println("Error:%v", err)
				}
				
				fmt.Println("PayEntryCBEntry JSON",  ":", string(buf.Bytes()))					

			} else 	if strings.Contains(t.String(), "BuyCBEntry"){
				fmt.Println("BuyCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err!=nil{
					fmt.Println("Error:%v", err)
				}			
				fmt.Println("BuyCBEntry JSON",  ":", string(buf.Bytes()))					
			}			
		}
		
		if err != nil {
	
			fmt.Println("Error:%v", err)
		}
	}

}

// Initialize the export file list
func initServerDataFileMap() error {
	serverDataFileMap = make(map[string]string)	
	
	fiList, err := ioutil.ReadDir(dataStorePath +"csv")
	if err != nil {
		fmt.Println("Error in initServerDataFileMap:", err.Error())
		return err
	}

	for _, file := range fiList{
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			hash := notaryapi.Sha([]byte(file.Name()))
				
			serverDataFileMap[hash.String()] = file.Name()

		}
	}	
	return nil	
	
}


func getServerDataFileMapJSON() (interface{}, *notaryapi.Error) {
	buf := new(bytes.Buffer)
	err := factomapi.SafeMarshal(buf, serverDataFileMap)

	var e *notaryapi.Error
	if err!=nil{	
		e = notaryapi.CreateError(notaryapi.ErrorBadPOSTData, err.Error())
	}
	return buf.Bytes(), e
}