// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// factomlog is based on github.com/alexcesaro/log and
// github.com/alexcesaro/log/golog (MIT License)

package restapi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/consensus"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/old/oldcoin"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/btcrpcclient"
	"github.com/FactomProject/btcutil"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
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
	db          database.Db       // database
	dchain      *notaryapi.DChain //Directory Block Chain
	cchain      *notaryapi.CChain //Entry Credit Chain
	fchain      *oldcoin.FChain   //Factoid Chain

	creditsPerChain   int32  = 10
	creditsPerFactoid uint64 = 1000

	// To be moved to ftmMemPool??
	chainIDMap      map[string]*notaryapi.EChain // ChainIDMap with chainID string([32]byte) as key
	eCreditMap      map[string]int32             // eCreditMap with public key string([32]byte) as key, credit balance as value
	prePaidEntryMap map[string]int32             // Paid but unrevealed entries string(Etnry Hash) as key, Number of payments as value

	chainIDMapBackup      map[string]*notaryapi.EChain //previous block bakcup - ChainIDMap with chainID string([32]byte) as key
	eCreditMapBackup      map[string]int32             // backup from previous block - eCreditMap with public key string([32]byte) as key, credit balance as value
	prePaidEntryMapBackup map[string]int32             // backup from previous block - Paid but unrevealed entries string(Etnry Hash) as key, Number of payments as value

	//Diretory Block meta data map
	dbInfoMap map[string]*notaryapi.DBInfo // dbInfoMap with dbHash string([32]byte) as key

	// to be renamed??
	inMsgQueue  chan wire.FtmInternalMsg //incoming message queue for factom application messages
	outMsgQueue chan wire.FtmInternalMsg //outgoing message queue for factom application messages

	//	intInMsgQueue  <-chan wire.FtmInternalMsg //incoming message queue for factom internal messages (from Factoid module to Factomd module)
	//	intOutMsgQueue chan<- wire.FtmInternalMsg //outgoing message queue for factom internal messages	(from Factomd module to Factoid module)

	fMemPool *ftmMemPool
	plMgr    *consensus.ProcessListMgr
)

var (
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

func LoadConfigurations(cfg *util.FactomdConfig) {

	//setting the variables by the valued form the config file
	logLevel = cfg.Log.LogLevel
	portNumber = cfg.App.PortNumber
	dataStorePath = cfg.App.DataStorePath
	ldbpath = cfg.App.LdbPath
	directoryBlockInSeconds = cfg.App.DirectoryBlockInSeconds
	nodeMode = cfg.App.NodeMode

	//addrStr = cfg.Btc.BTCPubAddr
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

	// create EBlocks and FBlock every 60 seconds
	tickers[0] = time.NewTicker(time.Second * time.Duration(directoryBlockInSeconds))

	// write 10 FBlock in a batch to BTC every 10 minutes
	tickers[1] = time.NewTicker(time.Second * time.Duration(sendToBTCinSeconds))

	go func() {
		for _ = range tickers[0].C {
			fmt.Println("in tickers[0]: newEntryBlock & newFactomBlock")

			eom10 := &wire.MsgInt_EOM{
				EOM_Type: wire.END_MINUTE_10,
			}

			inMsgQueue <- eom10

			/*
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

				util.Trace("NOT IMPLEMENTED: Factoid Chain init was here !!!!!!!!!!!")

				/*
					// Factoid Chain
					fBlock := newFBlock(fchain)
					if fBlock != nil {
						dchain.AddFBlockToDBEntry(factoid.NewDBEntryFromFBlock(fBlock))
					}
					saveFChain(fchain)
				*\

				// Directory Block chain
				dbBlock := newDirectoryBlock(dchain)
				saveDChain(dchain)

				// Only Servers can write the anchor to Bitcoin network
				if nodeMode == SERVER_NODE && dbBlock != nil {
					dbInfo := notaryapi.NewDBInfoFromDBlock(dbBlock)
					saveDBMerkleRoottoBTC(dbInfo)
				}

			*/
		}
	}()

}

func Start_Processor(ldb database.Db, inMsgQ chan wire.FtmInternalMsg, outMsgQ chan wire.FtmInternalMsg) {
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
	// Process msg from the incoming queue one by one
	for msg := range inMsgQ {
		fmt.Printf("in range inMsgQ, msg:%+v\n", msg)
		err := serveMsgRequest(msg)
		if err != nil {
			log.Println(err)
		}
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

func serveMsgRequest(msg wire.FtmInternalMsg) error {

	util.Trace()

	switch msg.Command() {
	case wire.CmdCommitChain:
		msgCommitChain, ok := msg.(*wire.MsgCommitChain)
		if ok && msgCommitChain.IsValid() {
			err := processCommitChain(msgCommitChain)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdRevealChain:
		msgRevealChain, ok := msg.(*wire.MsgRevealChain)
		if ok {
			err := processRevealChain(msgRevealChain)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdCommitEntry:
		msgCommitEntry, ok := msg.(*wire.MsgCommitEntry)
		if ok && msgCommitEntry.IsValid() {
			err := processCommitEntry(msgCommitEntry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdRevealEntry:
		msgRevealEntry, ok := msg.(*wire.MsgRevealEntry)
		if ok {
			err := processRevealEntry(msgRevealEntry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdInt_FactoidObj:
		return errors.New("TX type unsupported:" + fmt.Sprintf("%+v", msg) + "MUST BE REDONE !!!!!!!!!!!!!!!") // FIXME
		/*
			wireMsgTx, ok := msg.(*wire.MsgTx)
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
		*/
	case wire.CmdInt_EOM:
		msgEom, ok := msg.(*wire.MsgInt_EOM)
		if ok && msgEom.EOM_Type == wire.END_MINUTE_10 {
			plMgr.AddProcessListItem(msgEom, nil, wire.END_MINUTE_10)

			err := buildFromProcessListMgr()
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in build blocks:" + fmt.Sprintf("%+v", msg))
		}
	default:
		return errors.New("Message type unsupported:" + fmt.Sprintf("%+v", msg))
	}
	return nil
}

// Process a reveal-entry message and put it in the mem pool and the process list
// Put the message in the orphan pool if the message is out of order
func processRevealEntry(msg *wire.MsgRevealEntry) error {

	// Calculate the hash
	entryBinary, _ := msg.Entry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	shaHash, _ := wire.NewShaHash(entryHash.Bytes)

	chain := chainIDMap[msg.Entry.ChainID.String()]
	if chain == nil {
		fMemPool.addOrphanMsg(msg, shaHash)
		procLog.Debug("This chain is not supported:" + msg.Entry.ChainID.String())
	}

	// Calculate the required credits
	credits := int32(binary.Size(entryBinary)/1000 + 1)

	// Precalculate the key for prePaidEntryMap
	key := entryHash.String()

	// Delete the entry in the prePaidEntryMap in memory
	prepayment, ok := prePaidEntryMap[key]
	if !ok || prepayment < credits {
		fMemPool.addOrphanMsg(msg, shaHash)
		procLog.Debug("Credit needs to paid first before an entry is revealed:" + entryHash.String())
	}

	delete(prePaidEntryMap, key) // Only revealed once for multiple prepayments??

	// Add the msg to the Mem pool
	fMemPool.addMsg(msg, shaHash)

	// Add to MyPL if Server Node
	if nodeMode == SERVER_NODE {
		err := plMgr.AddProcessListItem(msg, shaHash, wire.ACK_REVEAL_ENTRY)
		if err != nil {
			return err
		}
	}

	return nil
}

// Process a commint-entry message and put it in the mem pool and the process list
// Put the message in the orphan pool if the message is out of order
func processCommitEntry(msg *wire.MsgCommitEntry) error {

	shaHash, _ := msg.Sha()

	// Update the credit balance in memory
	creditBalance, _ := eCreditMap[msg.ECPubKey.String()]
	if creditBalance < int32(msg.Credits) {
		fMemPool.addOrphanMsg(msg, &shaHash)
		procLog.Debug("Not enough credit for public key:" + msg.ECPubKey.String() + " Balance:" + fmt.Sprint(creditBalance))
	}
	eCreditMap[msg.ECPubKey.String()] = creditBalance - int32(msg.Credits)
	// Update the prePaidEntryMapin memory
	payments, _ := prePaidEntryMap[msg.EntryHash.String()]
	prePaidEntryMap[msg.EntryHash.String()] = payments + int32(msg.Credits)

	// Add to MyPL if Server Node
	if nodeMode == SERVER_NODE {
		err := plMgr.AddProcessListItem(msg, &shaHash, wire.ACK_COMMIT_ENTRY)
		if err != nil {
			return err
		}

	}
	return nil
}

func processCommitChain(msg *wire.MsgCommitChain) error {

	shaHash, _ := msg.Sha()

	// Check if the chain id already exists
	_, existing := chainIDMap[msg.ChainID.String()]
	if !existing {
		if msg.ChainID.IsSameAs(dchain.ChainID) || msg.ChainID.IsSameAs(cchain.ChainID) {
			existing = true
		}
	}
	if existing {
		return errors.New("Already existing chain id:" + msg.ChainID.String())
	}

	// Precalculate the key and value pair for prePaidEntryMap
	key := getPrePaidChainKey(msg.EntryHash, msg.ChainID)

	// Update the credit balance in memory
	creditBalance, _ := eCreditMap[msg.ECPubKey.String()]
	if creditBalance < int32(msg.Credits) {
		return errors.New("Insufficient credits for public key:" + msg.ECPubKey.String() + " Balance:" + fmt.Sprint(creditBalance))
	}
	eCreditMap[msg.ECPubKey.String()] = creditBalance - int32(msg.Credits)

	// Update the prePaidEntryMap in memory
	payments, _ := prePaidEntryMap[key]
	prePaidEntryMap[key] = payments + int32(msg.Credits)

	// Add to MyPL if Server Node
	if nodeMode == SERVER_NODE {
		err := plMgr.AddProcessListItem(msg, &shaHash, wire.ACK_COMMIT_CHAIN)
		if err != nil {
			return err
		}

	}

	return nil
}

func processBuyEntryCredit(pubKey *notaryapi.Hash, credits int32, factoidTxHash *notaryapi.Hash) error {

	// Update the credit balance in memory
	balance, _ := eCreditMap[pubKey.String()]
	eCreditMap[pubKey.String()] = balance + credits

	return nil
}

func processRevealChain(msg *wire.MsgRevealChain) error {
	shaHash, _ := msg.Sha()
	newChain := msg.Chain

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
		fMemPool.addOrphanMsg(msg, &shaHash)
		return errors.New("Enough credits need to paid first before creating a new chain:" + newChain.ChainID.String())
	}

	// Add the new chain in the chainIDMap
	chainIDMap[newChain.ChainID.String()] = newChain

	// Add to MyPL if Server Node
	if nodeMode == SERVER_NODE {
		err := plMgr.AddProcessListItem(msg, &shaHash, wire.ACK_REVEAL_CHAIN)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildRevealEntry(msg *wire.MsgRevealEntry) error {

	chain := chainIDMap[msg.Entry.ChainID.String()]

	// store the new entry in db
	entryBinary, _ := msg.Entry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	db.InsertEntryAndQueue(entryHash, &entryBinary, msg.Entry, &chain.ChainID.Bytes)

	err := chain.Blocks[len(chain.Blocks)-1].AddEBEntry(msg.Entry)

	if err != nil {
		return errors.New("Error while adding Entity to Block:" + err.Error())
	}

	return nil
}

func buildCommitEntry(msg *wire.MsgCommitEntry) error {

	// Create PayEntryCBEntry
	cbEntry := notaryapi.NewPayEntryCBEntry(msg.ECPubKey, msg.EntryHash, int32(0-msg.Credits), int64(msg.Timestamp))

	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)

	return err
}

func buildCommitChain(msg *wire.MsgCommitChain) error {

	// Create PayChainCBEntry
	cbEntry := notaryapi.NewPayChainCBEntry(msg.ECPubKey, msg.EntryHash, int32(0-msg.Credits), msg.ChainID, msg.EntryChainIDHash)

	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)

	return err
}

func buildBuyEntryCredit(pubKey *notaryapi.Hash, credits int32, factoidTxHash *notaryapi.Hash) error {

	cbEntry := notaryapi.NewBuyCBEntry(pubKey, factoidTxHash, credits)
	err := cchain.Blocks[len(cchain.Blocks)-1].AddCBEntry(cbEntry)

	return err
}

func buildRevealChain(msg *wire.MsgRevealChain) error {

	newChain := msg.Chain
	// Store the new chain in db
	db.InsertChain(newChain)

	// Chain initialization
	initEChainFromDB(newChain)
	fmt.Println("Loaded", len(newChain.Blocks)-1, "blocks for chain: "+newChain.ChainID.String())

	// store the new entry in db
	entryBinary, _ := newChain.FirstEntry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	db.InsertEntryAndQueue(entryHash, &entryBinary, newChain.FirstEntry, &newChain.ChainID.Bytes)

	err := newChain.Blocks[len(newChain.Blocks)-1].AddEBEntry(newChain.FirstEntry)

	if err != nil {
		return errors.New(fmt.Sprintf(`Error while adding the First Entry to Block: %s`, err.Error()))
	}

	return nil
}

// build blocks from all process lists
func buildFromProcessListMgr() error {

	if plMgr.MyProcessList.IsValid() {
		buildFromProcessList(plMgr.MyProcessList)
	}

	return nil
}

// build blocks from a process lists
func buildFromProcessList(pl *consensus.ProcessList) error {
	plItems := pl.GetPLItems()
	for _, pli := range plItems {
		if pli.Ack.Type == wire.ACK_COMMIT_CHAIN {
			buildCommitChain(pli.Msg.(*wire.MsgCommitChain))
		} else if pli.Ack.Type == wire.ACK_COMMIT_ENTRY {
			buildCommitEntry(pli.Msg.(*wire.MsgCommitEntry))
		} else if pli.Ack.Type == wire.ACK_REVEAL_CHAIN {
			buildRevealChain(pli.Msg.(*wire.MsgRevealChain))
		} else if pli.Ack.Type == wire.ACK_REVEAL_ENTRY {
			buildRevealEntry(pli.Msg.(*wire.MsgRevealEntry))
		} else if pli.Ack.Type == wire.ACK_FACTOID_TX {
			//buildCommitChain(pli.Msg.(*wire.MsgCommitEntry))??
		}
	}

	return nil
}

func newEntryBlock(chain *notaryapi.EChain) *notaryapi.EBlock {

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

	if len(block.EBEntries) < 1 {
		//log.Println("No new entry found. No block created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))
		return nil
	}

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	blkhash, _ := notaryapi.CreateHash(block)
	log.Println("blkhash:%v", blkhash.Bytes)
	block.IsSealed = true
	chain.NextBlockID++
	newblock, _ := notaryapi.CreateBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
	block.EBHash = blkhash

	// Create the Entry Block Merkle Root for FB Entry
	hashes := make([]*notaryapi.Hash, 0, len(block.EBEntries)+1)
	for _, entry := range block.EBEntries {
		data, _ := entry.MarshalBinary()
		hashes = append(hashes, notaryapi.Sha(data))
	}
	binaryEBHeader, _ := block.Header.MarshalBinary()
	hashes = append(hashes, notaryapi.Sha(binaryEBHeader))
	merkle := notaryapi.BuildMerkleTreeStore(hashes)
	block.MerkleRoot = merkle[len(merkle)-1] // MerkleRoot is not marshalized in Entry Block
	fmt.Println("block.MerkleRoot:%v", block.MerkleRoot.String())

	//Store the block in db
	db.ProcessEBlockBatch(block)
	log.Println("EntryBlock: block" + strconv.FormatUint(block.Header.BlockID, 10) + " created for chain: " + chain.ChainID.String())

	return block
}

func newEntryCreditBlock(chain *notaryapi.CChain) *notaryapi.CBlock {

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

	if len(block.CBEntries) < 1 {
		//log.Println("No new entry found. No block created for chain: "  + notaryapi.EncodeChainID(chain.ChainID))
		return nil
	}

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	block.Header.EntryCount = uint32(len(block.CBEntries))
	blkhash, _ := notaryapi.CreateHash(block)
	log.Println("blkhash:%v", blkhash.Bytes)
	block.IsSealed = true
	chain.NextBlockID++
	newblock, _ := notaryapi.CreateCBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()
	block.CBHash = blkhash

	//Store the block in db
	db.ProcessCBlockBatch(block)
	log.Println("EntryCreditBlock: block" + strconv.FormatUint(block.Header.BlockID, 10) + " created for chain: " + chain.ChainID.String())

	return block
}

func newFBlock(chain *oldcoin.FChain) *oldcoin.FBlock {

	// acquire the last block
	//	block := chain.Blocks[len(chain.Blocks)-1]

	//	if len(block.Transactions) < 1 {
	//log.Println("No Directory block created for chain ... because no new entry is found.")

	util.Trace("NOT IMPLEMENETED !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	return nil
	//	}

	/*
		// Create the block and add a new block for new coming entries
		chain.BlockMutex.Lock()
		block.Header.TxCount = uint32(len(block.Transactions))
		blkhash, _ := notaryapi.CreateHash(block)
		block.IsSealed = true
		chain.NextBlockID++
		newblock, _ := factoid.CreateFBlock(chain, block, 10)
		chain.Blocks = append(chain.Blocks, newblock)
		chain.BlockMutex.Unlock()
		block.FBHash = blkhash

		//Store the block in db
		db.ProcessFBlockBatch(block)
		log.Println("Factoid block" + strconv.FormatUint(block.Header.Height, 10) + " created for chain: " + chain.ChainID.String())

		return block
	*/
}

func newDirectoryBlock(chain *notaryapi.DChain) *notaryapi.DBlock {

	// acquire the last block
	block := chain.Blocks[len(chain.Blocks)-1]

	if len(block.DBEntries) < 1 {
		//log.Println("No Directory block created for chain ... because no new entry is found.")
		return nil
	}

	// Create the block add a new block for new coming entries
	chain.BlockMutex.Lock()
	block.Header.EntryCount = uint32(len(block.DBEntries))
	// Calculate Merkle Root for FBlock and store it in header
	if block.Header.MerkleRoot == nil {
		block.Header.MerkleRoot = block.CalculateMerkleRoot()
		fmt.Println("block.Header.MerkleRoot:%v", block.Header.MerkleRoot.String())
	}
	blkhash, _ := notaryapi.CreateHash(block)
	block.IsSealed = true
	chain.NextBlockID++
	newblock, _ := notaryapi.CreateDBlock(chain, block, 10)
	chain.Blocks = append(chain.Blocks, newblock)
	chain.BlockMutex.Unlock()

	//Store the block in db
	block.DBHash = blkhash
	db.ProcessDBlockBatch(block)

	log.Println("DirectoryBlock: block" + strconv.FormatUint(block.Header.BlockID, 10) + " created for directory block chain: " + chain.ChainID.String())

	return block
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

func saveFChain(chain *oldcoin.FChain) {
	util.Trace("NOT IMPLEMENTED !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	/*
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
	*/
}

func initDChain() {
	dchain = new(notaryapi.DChain)

	//Initialize dbInfoMap
	dbInfoMap = make(map[string]*notaryapi.DBInfo)

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

	// create a backup copy before processing entries
	copyCreditMap(eCreditMap, eCreditMapBackup)

}

// Initialize factoid chain from db and initialize the mempool??
func initFChain() {

	//Initialize the Entry Credit Chain ID
	fchain = new(oldcoin.FChain)
	barray := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	fchain.ChainID = new(notaryapi.Hash)
	fchain.ChainID.SetBytes(barray)

	util.Trace("NOT IMPLEMENTED: FIXME !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!") // FIXME

	/*
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

			util.Trace("CODE DISABLED !!!! MUST BE REDONE !!!!") // FIXME

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
	*/
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
func initProcessListMgr() {
	plMgr = consensus.NewProcessListMgr(dchain.NextBlockID, 1, 10)

}

func getPrePaidChainKey(entryHash *notaryapi.Hash, chainIDHash *notaryapi.Hash) string {
	return chainIDHash.String() + entryHash.String()
}

func copyCreditMap(originalMap map[string]int32, newMap map[string]int32) {

	// clean up the new map
	if newMap != nil {
		for k, _ := range newMap {
			delete(newMap, k)
		}
	} else {
		newMap = make(map[string]int32)
	}

	// copy every element from the original map
	for k, v := range originalMap {
		newMap[k] = v
	}

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
