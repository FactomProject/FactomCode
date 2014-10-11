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
	"reflect"
	"strconv"
	"strings"
	"os"
	"time"
	"log"
	"encoding/binary"
 
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/database/ldb"	

)

var client *btcrpcclient.Client
var currentAddr btcutil.Address
var balance int64

var portNumber = flag.Int("p", 8083, "Set the port to listen on")

var tickers [2]*time.Ticker

// database
var dbpath = "/tmp/ldb9"
var db database.Db


// ChainMap with string([32]byte) as key
var chainMap map[string]*notaryapi.Chain

//Factom Chain
var fchain *notaryapi.FChain

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func initWithBinary(chain *notaryapi.Chain) {
	matches, err := filepath.Glob("/tmp/store/seed/" + notaryapi.EncodeChainID(chain.ChainID) + "/store.*.block") // need to get it from a property file??
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
		panic ("chain.Blocks[chain.NextBlockID].IsSealed for chain:" + notaryapi.EncodeChainID(chain.ChainID))
	}
	chain.Blocks[chain.NextBlockID].EBEntries, _ = db.FetchEBEntriesFromQueue(chain.ChainID, &binaryTimestamp)		
}

func initDB() {
	
	//init db
	var err error
	db, err = ldb.OpenLevelDB(dbpath, false)
	
	if err != nil{
		log.Println("err opening db: %v", err)
	}
	
	if db == nil{
		db, err = ldb.OpenLevelDB(dbpath, true)
	}
	if err!=nil{
		panic(err)
	} else{
		log.Println("Database started from: " + dbpath)
	}
}

func init() { 
	gobundle.Setup.Application.Name = "Factom/restapi"
	gobundle.Init()
	
	initChainIDs() //for testing??
	
	initDB()
	
	dynrsrc.Start(watchError, readError)
	notaryapi.StartDynamic(gobundle.DataFile("html.gwp"), readError)
	
	for _, chain := range chainMap {
		initWithBinary(chain)
			
		fmt.Println("Loaded", len(chain.Blocks)-1, "blocks for chain: " + notaryapi.EncodeChainID(chain.ChainID))
	
		for i := 0; i < len(chain.Blocks); i = i + 1 {
			if uint64(i) != chain.Blocks[i].Header.BlockID {
				fmt.Println ("i:%v", i)
				fmt.Println ("chain.Blocks[i].Header.BlockID:%v", chain.Blocks[i].Header.BlockID)
				//bug to fix: store.10.block will come right after store.1.block
				panic(errors.New("BlockID does not equal index"))
			}
		}
	
	}


	// init FactomChain
	initFChain()
	fmt.Println("Loaded", len(fchain.Blocks)-1, "Factom blocks for chain: "+ notaryapi.EncodeChainID(fchain.ChainID))


	tickers[0] = time.NewTicker(time.Minute * 5)

	tickers[1] = time.NewTicker(time.Second * 30) 

	go func() {
		for _ = range tickers[1].C {
			for _, chain := range chainMap {
				eblock, blkhash := newEntryBlock(chain)
				if eblock != nil{
					fchain.AddFBEntry(eblock, blkhash)
				}
				save(chain)
			}
			newFactomBlock(fchain)
			saveFChain(fchain)		
			
		/*for testing	
			for _, chain := range chainMap {
				if len(chain.Blocks) < 2{
					continue
				}
				fmt.Println("Print out block ", len(chain.Blocks)-2, " for chain: " + notaryapi.EncodeChainID(chain.ChainID))
				block := chain.Blocks[chain.NextBlockID - 1]
				entryIB, _ := db.FetchEntryInfoBranchByHash((*block).EBEntries[0].Hash())
				
				fmt.Println(entryIB.EntryHash)
				
				if entryIB.EBInfo != nil{
					fmt.Println("entryIB.EBInfo.EBHash: %v", entryIB.EBInfo.EBHash)
				}
				if entryIB.FBInfo != nil{
					fmt.Println("entryIB.FBInfo.BTCTxHash: %v", entryIB.FBInfo.BTCTxHash)
				}


			}
			------------------------------------*/
							
		}

	}()
}


func main() {
	
	//addrStr := "muhXX7mXoMZUBvGLCgfjuoY2n2mziYETYC"
	addrStr := "movaFTARmsaTMk3j71MpX8HtMURpsKhdra"
	
	err := initRPCClient()
	if err != nil {
		log.Fatalf("cannot init rpc client: %s", err)
	}
	defer shutdown(client)
	
	if err := initWallet(addrStr); err != nil {
		log.Fatalf("cannot init wallet: %s", err)
	}
	
	
	flag.Parse()

	defer func() {
		tickers[0].Stop()
		tickers[1].Stop()
		dynrsrc.Stop()
		db.Close()
	}()

	http.HandleFunc("/", serveRESTfulHTTP)
	err = http.ListenAndServe(":"+strconv.Itoa(*portNumber), nil)
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
		log.Println("no blocks to save for chain: " + string (*chain.ChainID))
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

		strChainID := notaryapi.EncodeChainID(chain.ChainID)
		if fileNotExists ("/tmp/store/seed/" + strChainID){
			err:= os.MkdirAll("/tmp/store/seed/" + strChainID, 0777)
			if err==nil{
				log.Println("Created directory /tmp/store/seed/" + strChainID)
			} else{
				log.Println(err)
			}
		}

		err = ioutil.WriteFile(fmt.Sprintf("/tmp/store/seed/" + strChainID + "/store.%09d.block", i), data, 0777)
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

		buf.WriteTo(w)
		w.Write([]byte("\n\n"))
	}()

	switch method {
	case "GET":
		//resource, err = find(path)

	case "POST":
		if len(path) != 1 {
			err = notaryapi.CreateError(notaryapi.ErrorBadMethod, `POST can only be used in the root context: /v1`)
			return
		}

		resource, err = post("/"+strings.Join(path, "/"), form)

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

var blockPtrType = reflect.TypeOf((*notaryapi.Block)(nil)).Elem()

func post(context string, form url.Values) (interface{}, *notaryapi.Error) {
	newEntry := new(notaryapi.Entry)
	format, data, chainid := form.Get("format"), form.Get("data"), form.Get("chainid")

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

	default:
		return nil, notaryapi.CreateError(notaryapi.ErrorUnsupportedUnmarshal, fmt.Sprintf(`The format "%s" is not supported`, format))
	}

	if newEntry == nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Entity to be POSTed is nil`)
	}

	newEntry.StampTime()
	

	chainID, err1 := notaryapi.DecodeChainID(&chainid) //need a validation method??
	if err1 != nil{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `Not able to decode chain id`) //ErrorInternal?
	}
	chain := chainMap[string(chainID)]
	
	if chain == nil{
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, `This chain is not supported`) //ErrorInternal?
	}
	
	// store the new entry in db
	if db !=nil{
		entryBinary, _ := newEntry.MarshalBinary()
		
		hash, _ := notaryapi.CreateHash(newEntry)
		
		db.InsertEntryAndQueue( hash, &entryBinary, newEntry, &chainID)
	 
	} else{
		panic("db is null!")
	}

	chain.BlockMutex.Lock()
	err := chain.Blocks[len(chain.Blocks)-1].AddEBEntry(newEntry)
	chain.BlockMutex.Unlock()

	if err != nil {
		return nil, notaryapi.CreateError(notaryapi.ErrorInternal, fmt.Sprintf(`Error while adding Entity to Block: %s`, err.Error()))
	}

	return newEntry, nil
}


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

		strChainID := notaryapi.EncodeChainID(chain.ChainID)
		if fileNotExists ("/tmp/store/seed/" + strChainID){
			err:= os.MkdirAll("/tmp/store/seed/" + strChainID, 0777)
			if err==nil{
				log.Println("Created directory /tmp/store/seed/" + strChainID)
			} else{
				log.Println(err)
			}
		}
		err = ioutil.WriteFile(fmt.Sprintf("/tmp/store/seed/" + strChainID + "/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func initFChain() {
	fchain = new (notaryapi.FChain)

	barray := (make([]byte, 32))
	fchain.ChainID = &barray
	
	matches, err := filepath.Glob("/tmp/store/seed/" + notaryapi.EncodeChainID(fchain.ChainID) + "/store.*.block") // need to get it from a property file??
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
		panic ("fchain.Blocks[fchain.NextBlockID].IsSealed for chain:" + notaryapi.EncodeChainID(fchain.ChainID))
	}
	fchain.Blocks[fchain.NextBlockID].FBEntries, _ = db.FetchFBEntriesFromQueue(&binaryTimestamp)			

}

//for testing - to be moved into db-------------------------------------

var chainIDs [][]byte

func initChainIDs() {
	chainMap = make(map[string]*notaryapi.Chain)

	barray1 := make([]byte, 32)
	barray1[0] = byte(1)
	chainIDs = [][]byte{barray1}
	chain1 := new (notaryapi.Chain)
	chain1.ChainID = &chainIDs[0]
	chainMap[string(chainIDs[0])] = chain1
	
	barray2 := make([]byte, 32)
	barray2[0] = byte(2)
	chainIDs = append(chainIDs, barray2)
	chain2 := new (notaryapi.Chain)
	chain2.ChainID = &chainIDs[1]
	chainMap[string(chainIDs[1])] = chain2	

}
//--------------------------------------------------------------------------