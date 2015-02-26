package restapi

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/util"
	//"github.com/FactomProject/FactomCode/wallet"
	"github.com/FactomProject/btcrpcclient"
	"github.com/FactomProject/btcutil"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	//Server running mode
	FULL_NODE   = "FULL"
	SERVER_NODE = "SERVER"
	LIGHT_NODE  = "LIGHT"
)

var (
	wclient *btcrpcclient.Client //rpc client for btcwallet rpc server
	dclient *btcrpcclient.Client //rpc client for btcd rpc server

	currentAddr btcutil.Address
	tickers     [2]*time.Ticker
	db          database.Db                  // database
	chainIDMap  map[string]*notaryapi.EChain // ChainIDMap with chainID string([32]byte) as key
	//chainNameMap map[string]*notaryapi.Chain // ChainNameMap with chain name string as key
	dchain *notaryapi.DChain //Directory Block Chain
	cchain *notaryapi.CChain //Entry Credit Chain
	fchain *factoid.FChain   //Factoid Chain

	creditsPerChain   int32            = 10
	creditsPerFactoid uint64           = 1000
	eCreditMap        map[string]int32 // eCreditMap with public key string([32]byte) as key, credit balance as value
	prePaidEntryMap   map[string]int32 // Paid but unrevealed entries string(Etnry Hash) as key, Number of payments as value

	//	dbBatches []*notaryapi.FBBatch
	dbBatches *DBBatches
	dbBatch   *notaryapi.DBBatch

	//Map to store export csv files
	ServerDataFileMap map[string]string

	inMsgQueue  <-chan factomwire.Message //incoming message queue for factom application messages
	outMsgQueue chan<- factomwire.Message //outgoing message queue for factom application messages
)

var (
	logLevel                    = "DEBUG"
	portNumber              int = 8083
	sendToBTCinSeconds          = 600
	directoryBlockInSeconds     = 60
	dataStorePath               = "/tmp/store/seed/"
	ldbpath                     = "/tmp/ldb9"
	nodeMode                    = "FULL"

	//BTC:
	//	addrStr = "movaFTARmsaTMk3j71MpX8HtMURpsKhdra"
	walletPassphrase          = "lindasilva"
	certHomePath              = "btcwallet"
	rpcClientHost             = "localhost:18332" //btcwallet rpcserver address
	rpcClientEndpoint         = "ws"
	rpcClientUser             = "testuser"
	rpcClientPass             = "notarychain"
	btcTransFee       float64 = 0.0001

	certHomePathBtcd = "btcd"
	rpcBtcdHost      = "localhost:18334" //btcd rpcserver address

)

type DBBatches struct {
	batches    []*notaryapi.DBBatch
	batchMutex sync.Mutex
}

func LoadConfigurations(cfg *util.FactomdConfig) {

	//setting the variables by the valued form the config file
	logLevel = cfg.Log.LogLevel
	portNumber = cfg.App.PortNumber
	dataStorePath = cfg.App.DataStorePath
	ldbpath = cfg.App.LdbPath
	directoryBlockInSeconds = cfg.App.DirectoryBlockInSeconds
	nodeMode = cfg.App.NodeMode

	//		addrStr = cfg.Btc.BTCPubAddr
	sendToBTCinSeconds = cfg.Btc.SendToBTCinSeconds
	walletPassphrase = cfg.Btc.WalletPassphrase
	certHomePath = cfg.Btc.CertHomePath
	rpcClientHost = cfg.Btc.RpcClientHost
	rpcClientEndpoint = cfg.Btc.RpcClientEndpoint
	rpcClientUser = cfg.Btc.RpcClientUser
	rpcClientPass = cfg.Btc.RpcClientPass
	btcTransFee = cfg.Btc.BtcTransFee
	certHomePathBtcd = cfg.Btc.CertHomePathBtcd
	rpcBtcdHost = cfg.Btc.RpcBtcdHost //btcd rpcserver address

}

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

// Initialize the entry chains in memory from db
func initEChainFromDB(chain *notaryapi.EChain) {

	eBlocks, _ := db.FetchAllEBlocksByChain(chain.ChainID)
	sort.Sort(util.ByEBlockIDAccending(*eBlocks))

	chain.Blocks = make([]*notaryapi.EBlock, len(*eBlocks))

	for i := 0; i < len(*eBlocks); i = i + 1 {
		if (*eBlocks)[i].Header.BlockID != uint64(i) {
			panic("Error in initializing EChain:" + chain.ChainID.String())
		}
		(*eBlocks)[i].Chain = chain
		(*eBlocks)[i].IsSealed = true
		chain.Blocks[i] = &(*eBlocks)[i]
	}

	//Create an empty block and append to the chain
	if len(chain.Blocks) == 0 {
		chain.NextBlockID = 0
		newblock, _ := notaryapi.CreateBlock(chain, nil, 10)

		chain.Blocks = append(chain.Blocks, newblock)
	} else {
		chain.NextBlockID = uint64(len(chain.Blocks))
		newblock, _ := notaryapi.CreateBlock(chain, chain.Blocks[len(chain.Blocks)-1], 10)
		chain.Blocks = append(chain.Blocks, newblock)
	}

	//Get the unprocessed entries in db for the past # of mins for the open block
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	if chain.Blocks[chain.NextBlockID].IsSealed == true {
		panic("chain.Blocks[chain.NextBlockID].IsSealed for chain:" + chain.ChainID.String())
	}
	chain.Blocks[chain.NextBlockID].EBEntries, _ = db.FetchEBEntriesFromQueue(&chain.ChainID.Bytes, &binaryTimestamp)
}

func init_processor() {

	// init Directory Block Chain
	initDChain()
	fmt.Println("Loaded", len(dchain.Blocks)-1, "Directory blocks for chain: "+dchain.ChainID.String())

	// init Entry Credit Chain
	initCChain()
	fmt.Println("Loaded", len(cchain.Blocks)-1, "Entry Credit blocks for chain: "+cchain.ChainID.String())

	// init Factoind Chain
	initFChain()
	fmt.Println("Loaded", len(fchain.Blocks)-1, "Factoid blocks for chain: "+fchain.ChainID.String())

	// init Entry Chains
	initEChains()
	for _, chain := range chainIDMap {
		initEChainFromDB(chain)

		fmt.Println("Loaded", len(chain.Blocks)-1, "blocks for chain: "+chain.ChainID.String())

		for i := 0; i < len(chain.Blocks); i = i + 1 {
			if uint64(i) != chain.Blocks[i].Header.BlockID {
				panic(errors.New("BlockID does not equal index for chain:" + chain.ChainID.String() + " block:" + fmt.Sprintf("%v", chain.Blocks[i].Header.BlockID)))
			}
		}

	}

	// init dbBatches, dbBatch
	dbBatches = &DBBatches{
		batches: make([]*notaryapi.DBBatch, 0, 100),
	}
	dbBatch := &notaryapi.DBBatch{
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
				if eblock != nil {
					dchain.AddDBEntry(eblock)
				}
				save(chain)
			}

			// Entry Credit Chain
			cBlock := newEntryCreditBlock(cchain)
			if cBlock != nil {
				dchain.AddCBlockToDBEntry(cBlock)
			}
			saveCChain(cchain)

			// Factoid Chain
			fBlock := newFBlock(fchain)
			if fBlock != nil {
				dchain.AddFBlockToDBEntry(factoid.NewDBEntryFromFBlock(fBlock))
			}
			saveFChain(fchain)

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
				dbBatch = &notaryapi.DBBatch{
					DBlocks: make([]*notaryapi.DBlock, 0, 10),
				}

				dbBatches.batchMutex.Lock()
				dbBatches.batches = append(dbBatches.batches, doneBatch)
				dbBatches.batchMutex.Unlock()

				fmt.Printf("in tickers[1]: doneBatch=%#v\n", doneBatch)

				// Only Servers can write the anchor to Bitcoin network
				if nodeMode == SERVER_NODE {
					saveDBBatchMerkleRoottoBTC(doneBatch)
				}
			}
		}
	}()

}

func Start_Processor(ldb database.Db, inMsgQ <-chan factomwire.Message, outMsgQ chan<- factomwire.Message) {

	db = ldb
	inMsgQueue = inMsgQ
	outMsgQueue = outMsgQ

	init_processor()

	if nodeMode == SERVER_NODE {
		err := initRPCClient()
		if err != nil {
			log.Fatalf("cannot init rpc client: %s", err)
		}

		if err := initWallet(); err != nil {
			log.Fatalf("cannot init wallet: %s", err)
		}
	}
	util.Trace("before range inMsgQ")
	// Process msg from the incoming queue
	for msg := range inMsgQ {
		fmt.Printf("in range inMsgQ, msg:%+v\n", msg)
		go serveMsgRequest(msg)
	}

	util.Trace()

	defer func() {
		shutdown()
		tickers[0].Stop()
		tickers[1].Stop()
		//db.Close()
	}()

}

func fileNotExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return true
	}
	return err != nil
}

func save(chain *notaryapi.EChain) {
	if len(chain.Blocks) == 0 {
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
		if fileNotExists(dataStorePath + strChainID) {
			err := os.MkdirAll(dataStorePath+strChainID, 0777)
			if err == nil {
				log.Println("Created directory " + dataStorePath + strChainID)
			} else {
				log.Println(err)
			}
		}

		err = ioutil.WriteFile(fmt.Sprintf(dataStorePath+strChainID+"/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func serveMsgRequest(msg factomwire.Message) error {
	//var buf bytes.Buffer

	util.Trace()

	switch msg.Command() {
	case factomwire.CmdCommitChain:
		msgCommitChain, ok := msg.(*factomwire.MsgCommitChain)
		if ok {
			//Verify signature (timestamp + chainid + entry hash + entryChainIDHash + credits)
			var buf bytes.Buffer
			binary.Write(&buf, binary.BigEndian, msgCommitChain.Timestamp)
			buf.Write(msgCommitChain.ChainID.Bytes)
			buf.Write(msgCommitChain.EntryHash.Bytes)
			buf.Write(msgCommitChain.EntryChainIDHash.Bytes)
			binary.Write(&buf, binary.BigEndian, msgCommitChain.Credits)
			if !notaryapi.VerifySlice(msgCommitChain.ECPubKey.Bytes, buf.Bytes(), msgCommitChain.Sig) {
				return errors.New("Error in verifying signature for msg:" + fmt.Sprintf("%+v", msgCommitChain))
			}

			err := processCommitChain(msgCommitChain.EntryHash, msgCommitChain.ChainID, msgCommitChain.EntryChainIDHash, msgCommitChain.ECPubKey, int32(msgCommitChain.Credits))
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case factomwire.CmdRevealChain:
		msgRevealChain, ok := msg.(*factomwire.MsgRevealChain)
		if ok {
			err := processRevealChain(msgRevealChain.Chain)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case factomwire.CmdCommitEntry:
		msgCommitEntry, ok := msg.(*factomwire.MsgCommitEntry)
		if ok {
			//Verify signature (timestamp + entry hash + credits)
			var buf bytes.Buffer
			binary.Write(&buf, binary.BigEndian, msgCommitEntry.Timestamp)
			buf.Write(msgCommitEntry.EntryHash.Bytes)
			binary.Write(&buf, binary.BigEndian, msgCommitEntry.Credits)
			if !notaryapi.VerifySlice(msgCommitEntry.ECPubKey.Bytes, buf.Bytes(), msgCommitEntry.Sig) {
				return errors.New("Error in verifying signature for msg:" + fmt.Sprintf("%+v", msgCommitEntry))
			}

			err := processCommitEntry(msgCommitEntry.EntryHash, msgCommitEntry.ECPubKey, int64(msgCommitEntry.Timestamp), int32(msgCommitEntry.Credits))
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}
		
	case factomwire.CmdRevealEntry:
		msgRevealEntry, ok := msg.(*factomwire.MsgRevealEntry)
		if ok {
			err := processRevealEntry(msgRevealEntry.Entry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}
		
	case factomwire.CmdTx:
		wireMsgTx, ok := msg.(*factomwire.MsgTx)
		if ok {
			txm := new(factoid.TxMsg)
			err := txm.UnmarshalBinary(wireMsgTx.Data)		
			if err != nil {
				return err
			}				
			err = processFactoidTx(factoid.NewTx(txm))
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}
		/*  There is no such command !
		case factomwire.CmdGetCredit:
			msgGetCredit, ok := msg.(*factomwire.MsgGetCredit)
			if ok {
				fmt.Printf("msgGetCredit:%+v\n", msgGetCredit)
				credits := msgGetCredit.FactoidBase * creditsPerFactoid / 1000000000
				err := processBuyEntryCredit(msgGetCredit.ECPubKey, int32(credits), msgGetCredit.ECPubKey)
				if err != nil {
					return err
				}
			} else {
				return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
			}
		*/ 

	default:
		return errors.New("Message type unsupported:" + fmt.Sprintf("%+v", msg))
	}
	return nil
}

// to be improved??
func processFactoidTx(tx *factoid.Tx) error {


	fchain.BlockMutex.Lock()
	err := fchain.Blocks[len(fchain.Blocks)-1].AddFBTransaction(*tx)
	fchain.BlockMutex.Unlock()

	if err != nil {
		return errors.New("Error while adding Factoid transaction to Block:" + err.Error())
	}

	return nil
}

func processRevealEntry(newEntry *notaryapi.Entry) error {

	chain := chainIDMap[newEntry.ChainID.String()]
	if chain == nil {
		return errors.New("This chain is not supported:" + newEntry.ChainID.String())
	}

	// store the new entry in db
	entryBinary, _ := newEntry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	db.InsertEntryAndQueue(entryHash, &entryBinary, newEntry, &chain.ChainID.Bytes)

	// Calculate the required credits
	credits := int32(binary.Size(entryBinary)/1000 + 1)

	// Precalculate the key for prePaidEntryMap
	key := entryHash.String()

	// Delete the entry in the prePaidEntryMap in memory
	prepayment, ok := prePaidEntryMap[key]
	if ok && prepayment >= credits {
		delete(prePaidEntryMap, key) // Only revealed once for multiple prepayments??
	} else {
		return errors.New("Credit needs to paid first before reveal an entry:" + entryHash.String())
	}

	chain.BlockMutex.Lock()
	err := chain.Blocks[len(chain.Blocks)-1].AddEBEntry(newEntry)
	chain.BlockMutex.Unlock()

	if err != nil {
		return errors.New("Error while adding Entity to Block:" + err.Error())
	}

	return nil
}

func processCommitEntry(entryHash *notaryapi.Hash, pubKey *notaryapi.Hash, timeStamp int64, credits int32) error {
	// Make sure credits is negative
	if credits > 0 {
		credits = 0 - credits
	}
	// Create PayEntryCBEntry
	cbEntry := notaryapi.NewPayEntryCBEntry(pubKey, entryHash, credits, timeStamp)

	cchain.BlockMutex.Lock()
	// Update the credit balance in memory
	creditBalance, _ := eCreditMap[pubKey.String()]
	if creditBalance+credits < 0 {
		cchain.BlockMutex.Unlock()
		return errors.New("Not enough credit for public key:" + pubKey.String() + " Balance:" + fmt.Sprint(credits))
	}
	eCreditMap[pubKey.String()] = creditBalance + credits
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)
	// Update the prePaidEntryMapin memory
	payments, _ := prePaidEntryMap[entryHash.String()]
	prePaidEntryMap[entryHash.String()] = payments - credits
	cchain.BlockMutex.Unlock()

	return err
}

func processCommitChain(entryHash *notaryapi.Hash, chainIDHash *notaryapi.Hash, entryChainIDHash *notaryapi.Hash, pubKey *notaryapi.Hash, credits int32) error {

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
		return errors.New("Already existing chain id:" + chainIDHash.String())
	}

	// Precalculate the key and value pair for prePaidEntryMap
	key := getPrePaidChainKey(entryHash, chainIDHash)

	// Create PayChainCBEntry
	cbEntry := notaryapi.NewPayChainCBEntry(pubKey, entryHash, credits, chainIDHash, entryChainIDHash)

	cchain.BlockMutex.Lock()
	// Update the credit balance in memory
	creditBalance, _ := eCreditMap[pubKey.String()]
	if creditBalance+credits < 0 {
		cchain.BlockMutex.Unlock()
		return errors.New("Insufficient credits for public key:" + pubKey.String() + " Balance:" + fmt.Sprint(credits))
	}
	eCreditMap[pubKey.String()] = creditBalance + credits
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)
	// Update the prePaidEntryMap in memory
	payments, _ := prePaidEntryMap[key]
	prePaidEntryMap[key] = payments - credits
	cchain.BlockMutex.Unlock()

	return err
}

func processBuyEntryCredit(pubKey *notaryapi.Hash, credits int32, factoidTxHash *notaryapi.Hash) error {

	cbEntry := notaryapi.NewBuyCBEntry(pubKey, factoidTxHash, credits)
	cchain.BlockMutex.Lock()
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)
	// Update the credit balance in memory
	balance, _ := eCreditMap[pubKey.String()]
	eCreditMap[pubKey.String()] = balance + credits
	cchain.BlockMutex.Unlock()

	printCChain()
	printCreditMap()
	return err
}

func processRevealChain(newChain *notaryapi.EChain) error {
	// Check if the chain id already exists
	_, existing := chainIDMap[newChain.ChainID.String()]
	if !existing {
		if newChain.ChainID.IsSameAs(dchain.ChainID) || newChain.ChainID.IsSameAs(cchain.ChainID) {
			existing = true
		}
	}
	if existing {
		return errors.New("This chain is already existing:" + newChain.ChainID.String())
	}

	if newChain.FirstEntry == nil {
		return errors.New("The first entry is required to create a new chain:" + newChain.ChainID.String())
	}
	// Calculate the required credits
	binaryChain, _ := newChain.MarshalBinary()
	credits := int32(binary.Size(binaryChain)/1000+1) + creditsPerChain

	// Remove the entry for prePaidEntryMap
	binaryEntry, _ := newChain.FirstEntry.MarshalBinary()
	firstEntryHash := notaryapi.Sha(binaryEntry)
	key := getPrePaidChainKey(firstEntryHash, newChain.ChainID)
	prepayment, ok := prePaidEntryMap[key]
	if ok && prepayment >= credits {
		delete(prePaidEntryMap, key)
	} else {
		return errors.New("Enough credits need to paid first before creating a new chain:" + newChain.ChainID.String())
	}
	// Store the new chain in db
	db.InsertChain(newChain)

	// Chain initialization
	initEChainFromDB(newChain)
	fmt.Println("Loaded", len(newChain.Blocks)-1, "blocks for chain: "+newChain.ChainID.String())

	// Add the new chain in the chainIDMap
	chainIDMap[newChain.ChainID.String()] = newChain

	// store the new entry in db
	entryBinary, _ := newChain.FirstEntry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	db.InsertEntryAndQueue(entryHash, &entryBinary, newChain.FirstEntry, &newChain.ChainID.Bytes)

	newChain.BlockMutex.Lock()
	err := newChain.Blocks[len(newChain.Blocks)-1].AddEBEntry(newChain.FirstEntry)
	newChain.BlockMutex.Unlock()

	if err != nil {
		return errors.New(fmt.Sprintf(`Error while adding the First Entry to Block: %s`, err.Error()))
	}

	ExportDataFromDbToFile()

	return nil
}

func getEntryCreditBalance(pubKey *notaryapi.Hash) ([]byte, error) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, eCreditMap[pubKey.String()])
	return buf.Bytes(), nil
}

func saveDChain(chain *notaryapi.DChain) {
	if len(chain.Blocks) == 0 {
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
		if fileNotExists(dataStorePath + strChainID) {
			err := os.MkdirAll(dataStorePath+strChainID, 0777)
			if err == nil {
				log.Println("Created directory " + dataStorePath + strChainID)
			} else {
				log.Println(err)
			}
		}
		err = ioutil.WriteFile(fmt.Sprintf(dataStorePath+strChainID+"/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func saveCChain(chain *notaryapi.CChain) {
	if len(chain.Blocks) == 0 {
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
		if fileNotExists(dataStorePath + strChainID) {
			err := os.MkdirAll(dataStorePath+strChainID, 0777)
			if err == nil {
				log.Println("Created directory " + dataStorePath + strChainID)
			} else {
				log.Println(err)
			}
		}
		err = ioutil.WriteFile(fmt.Sprintf(dataStorePath+strChainID+"/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func saveFChain(chain *factoid.FChain) {
	if len(chain.Blocks) == 0 {
		//log.Println("no blocks to save for chain: " + string (*chain.ChainID))
		return
	}

	bcp := make([]*factoid.FBlock, len(chain.Blocks))

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
		if fileNotExists(dataStorePath + strChainID) {
			err := os.MkdirAll(dataStorePath+strChainID, 0777)
			if err == nil {
				log.Println("Created directory " + dataStorePath + strChainID)
			} else {
				log.Println(err)
			}
		}
		err = ioutil.WriteFile(fmt.Sprintf(dataStorePath+strChainID+"/store.%09d.block", i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}

func initDChain() {
	dchain = new(notaryapi.DChain)

	//Initialize the Directory Block Chain ID
	dchain.ChainID = new(notaryapi.Hash)
	barray := []byte{0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD,
		0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD}
	dchain.ChainID.SetBytes(barray)

	// get all dBlocks from db
	dBlocks, _ := db.FetchAllDBlocks()
	sort.Sort(util.ByDBlockIDAccending(dBlocks))

	dchain.Blocks = make([]*notaryapi.DBlock, len(dBlocks))

	for i := 0; i < len(dBlocks); i = i + 1 {
		if dBlocks[i].Header.BlockID != uint64(i) {
			panic("Error in initializing dChain:" + dchain.ChainID.String())
		}
		dBlocks[i].Chain = dchain
		dBlocks[i].IsSealed = true
		dchain.Blocks[i] = &dBlocks[i]
	}

	// double check the block ids
	for i := 0; i < len(dchain.Blocks); i = i + 1 {
		if uint64(i) != dchain.Blocks[i].Header.BlockID {
			panic(errors.New("BlockID does not equal index for chain:" + dchain.ChainID.String() + " block:" + fmt.Sprintf("%v", dchain.Blocks[i].Header.BlockID)))
		}
	}

	//Create an empty block and append to the chain
	if len(dchain.Blocks) == 0 {
		dchain.NextBlockID = 0
		newblock, _ := notaryapi.CreateDBlock(dchain, nil, 10)
		dchain.Blocks = append(dchain.Blocks, newblock)

	} else {
		dchain.NextBlockID = uint64(len(dchain.Blocks))
		newblock, _ := notaryapi.CreateDBlock(dchain, dchain.Blocks[len(dchain.Blocks)-1], 10)
		dchain.Blocks = append(dchain.Blocks, newblock)
	}

	//Get the unprocessed entries in db for the past # of mins for the open block
	binaryTimestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryTimestamp, uint64(0))
	if dchain.Blocks[dchain.NextBlockID].IsSealed == true {
		panic("dchain.Blocks[dchain.NextBlockID].IsSealed for chain:" + dchain.ChainID.String())
	}
	dchain.Blocks[dchain.NextBlockID].DBEntries, _ = db.FetchDBEntriesFromQueue(&binaryTimestamp)

}

func initCChain() {

	eCreditMap = make(map[string]int32)
	prePaidEntryMap = make(map[string]int32)

	//Initialize the Entry Credit Chain ID
	cchain = new(notaryapi.CChain)
	barray := []byte{0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC,
		0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC}
	cchain.ChainID = new(notaryapi.Hash)
	cchain.ChainID.SetBytes(barray)

	// get all dBlocks from db
	cBlocks, _ := db.FetchAllCBlocks()
	sort.Sort(util.ByCBlockIDAccending(cBlocks))

	cchain.Blocks = make([]*notaryapi.CBlock, len(cBlocks))

	for i := 0; i < len(cBlocks); i = i + 1 {
		if cBlocks[i].Header.BlockID != uint64(i) {
			panic("Error in initializing dChain:" + cchain.ChainID.String())
		}
		cBlocks[i].Chain = cchain
		cBlocks[i].IsSealed = true
		cchain.Blocks[i] = &cBlocks[i]

		// Calculate the EC balance for each account
		initializeECreditMap(cchain.Blocks[i])
	}

	// double check the block ids
	for i := 0; i < len(cchain.Blocks); i = i + 1 {
		if uint64(i) != cchain.Blocks[i].Header.BlockID {
			panic(errors.New("BlockID does not equal index for chain:" + cchain.ChainID.String() + " block:" + fmt.Sprintf("%v", cchain.Blocks[i].Header.BlockID)))
		}
	}

	//Create an empty block and append to the chain
	if len(cchain.Blocks) == 0 {
		cchain.NextBlockID = 0
		newblock, _ := notaryapi.CreateCBlock(cchain, nil, 10)
		cchain.Blocks = append(cchain.Blocks, newblock)

	} else {
		cchain.NextBlockID = uint64(len(cchain.Blocks))
		newblock, _ := notaryapi.CreateCBlock(cchain, cchain.Blocks[len(cchain.Blocks)-1], 10)
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

// Initialize factoid chain from db and initialize the mempool??
func initFChain() {

	//Initialize the Entry Credit Chain ID
	fchain = new(factoid.FChain)
	barray := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	fchain.ChainID = new(notaryapi.Hash)
	fchain.ChainID.SetBytes(barray)

	// get all dBlocks from db
	fBlocks, _ := db.FetchAllFBlocks()
	sort.Sort(factoid.ByFBlockIDAccending(fBlocks))

	fchain.Blocks = make([]*factoid.FBlock, len(fBlocks))

	for i := 0; i < len(fBlocks); i = i + 1 {
		if fBlocks[i].Header.Height != uint64(i) {
			panic("Error in initializing fChain:" + fchain.ChainID.String())
		}
		fBlocks[i].Chain = fchain
		fBlocks[i].IsSealed = true
		fchain.Blocks[i] = &fBlocks[i]

		// Load the block into utxo pool
		factoid.GlobalUtxo.AddVerifiedTxList(fBlocks[i].Transactions)
		//fmt.Printf("fchain.Blocks:%+v\n", fchain.Blocks[i])
	}

	// double check the block ids
	for i := 0; i < len(fchain.Blocks); i = i + 1 {
		if uint64(i) != fchain.Blocks[i].Header.Height {
			panic(errors.New("BlockID does not equal index for chain:" + fchain.ChainID.String() + " block:" + fmt.Sprintf("%v", fchain.Blocks[i].Header.Height)))
		}
	}

	//Create an empty block and append to the chain
	if len(fchain.Blocks) == 0 {
		fchain.NextBlockID = 0
		newblock, _ := factoid.CreateFBlock(fchain, nil, 10)
		fchain.Blocks = append(fchain.Blocks, newblock)

	} else {
		fchain.NextBlockID = uint64(len(fchain.Blocks))
		newblock, _ := factoid.CreateFBlock(fchain, fchain.Blocks[len(fchain.Blocks)-1], 10)
		fchain.Blocks = append(fchain.Blocks, newblock)
	}
}

func ExportDbToFile(dbHash *notaryapi.Hash) {

	if fileNotExists(dataStorePath + "csv/") {
		os.MkdirAll(dataStorePath+"csv/", 0755)
	}

	//write the records to a csv file:
	filename := fmt.Sprintf("%v."+dbHash.String()+".csv", time.Now().Unix())
	file, err := os.Create(dataStorePath + "csv/" + filename)

	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)

	ldbMap, err := db.FetchAllDBRecordsByDBHash(dbHash, cchain.ChainID)

	if err != nil {
		log.Println(err)
		return
	}

	for key, value := range ldbMap {
		//csv header: key, value
		writer.Write([]string{key, value})
	}
	writer.Flush()

	// Add the file to the distribution list
	hash := notaryapi.Sha([]byte(filename))
	ServerDataFileMap[hash.String()] = filename
}

func ExportDataFromDbToFile() {

	if fileNotExists(dataStorePath + "csv/") {
		os.MkdirAll(dataStorePath+"csv/", 0755)
	}

	//write the records to a csv file:
	filename := fmt.Sprintf("%v.supportdata.csv", time.Now().Unix())
	file, err := os.Create(dataStorePath + "csv/" + filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)

	ldbMap, err := db.FetchSupportDBRecords()

	if err != nil {
		log.Println(err)
		return
	}

	for key, value := range ldbMap {
		//csv header: key, value
		writer.Write([]string{key, value})
	}
	writer.Flush()

	// Add the file to the distribution list
	hash := notaryapi.Sha([]byte(filename))
	ServerDataFileMap[hash.String()] = filename

}

func initEChains() {

	chainIDMap = make(map[string]*notaryapi.EChain)

	chains, err := db.FetchAllChainsByName(nil)

	if err != nil {
		panic(err)
	}

	for _, chain := range *chains {
		var newChain = chain
		chainIDMap[newChain.ChainID.String()] = &newChain
	}

}

func initializeECreditMap(block *notaryapi.CBlock) {
	for _, cbEntry := range block.CBEntries {
		credits, _ := eCreditMap[cbEntry.PublicKey().String()]
		eCreditMap[cbEntry.PublicKey().String()] = credits + cbEntry.Credits()
	}
}

func getPrePaidChainKey(entryHash *notaryapi.Hash, chainIDHash *notaryapi.Hash) string {
	return chainIDHash.String() + entryHash.String()
}

func printCreditMap() {

	fmt.Println("eCreditMap:")
	for key := range eCreditMap {
		fmt.Println("Key:", key, "Value", eCreditMap[key])
	}
}

func printPaidEntryMap() {

	fmt.Println("prePaidEntryMap:")
	for key := range prePaidEntryMap {
		fmt.Println("Key:", key, "Value", prePaidEntryMap[key])
	}
}

func printCChain() {

	fmt.Println("cchain:", cchain.ChainID.String())

	for i, block := range cchain.Blocks {
		if !block.IsSealed {
			continue
		}
		var buf bytes.Buffer
		err := factomapi.SafeMarshal(&buf, block.Header)

		fmt.Println("block.Header", string(i), ":", string(buf.Bytes()))

		for _, cbentry := range block.CBEntries {
			t := reflect.TypeOf(cbentry)
			fmt.Println("cbEntry Type:", t.Name(), t.String())
			if strings.Contains(t.String(), "PayChainCBEntry") {
				fmt.Println("PayChainCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err != nil {
					fmt.Println("Error:%v", err)
				}

				fmt.Println("PayChainCBEntry JSON", ":", string(buf.Bytes()))

			} else if strings.Contains(t.String(), "PayEntryCBEntry") {
				fmt.Println("PayEntryCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err != nil {
					fmt.Println("Error:%v", err)
				}

				fmt.Println("PayEntryCBEntry JSON", ":", string(buf.Bytes()))

			} else if strings.Contains(t.String(), "BuyCBEntry") {
				fmt.Println("BuyCBEntry - pubkey:", cbentry.PublicKey().String(), " Credits:", cbentry.Credits())
				var buf bytes.Buffer
				err := factomapi.SafeMarshal(&buf, cbentry)
				if err != nil {
					fmt.Println("Error:%v", err)
				}
				fmt.Println("BuyCBEntry JSON", ":", string(buf.Bytes()))
			}
		}

		if err != nil {

			fmt.Println("Error:%v", err)
		}
	}

}

// Initialize the export file list
func initServerDataFileMap() error {
	ServerDataFileMap = make(map[string]string)

	fiList, err := ioutil.ReadDir(dataStorePath + "csv")
	if err != nil {
		fmt.Println("Error in initServerDataFileMap:", err.Error())
		return err
	}

	for _, file := range fiList {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			hash := notaryapi.Sha([]byte(file.Name()))

			ServerDataFileMap[hash.String()] = file.Name()

		}
	}
	return nil

}
