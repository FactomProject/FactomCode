// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.
// github.com/alexcesaro/log/golog (MIT License)

// Processor is the engine of Factom
// It processes all of the incoming messages from the network
// It syncs up with peers and build blocks based on the process lists and a timed schedule
// For details, please refer to:
// https://github.com/FactomProject/FactomDocs/blob/master/FactomLedgerbyConsensus.pdf

package process

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"

	"github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	"github.com/FactomProject/FactomCode/database"

	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd/wire"
	sc "github.com/FactomProject/factoid"
	"github.com/FactomProject/factoid/block"
	"github.com/davecgh/go-spew/spew"
)

var _ = (*sc.Transaction)(nil)
var _ = (*block.FBlock)(nil)

var (
	db       database.Db        // database
	dchain   *common.DChain     //Directory Block Chain
	ecchain  *common.ECChain    //Entry Credit Chain
	achain   *common.AdminChain //Admin Chain
	scchain  *common.FctChain    // factoid Chain
	fchainID *common.Hash
	
	inMsgQueue  chan wire.FtmInternalMsg //incoming message queue for factom application messages
	outMsgQueue chan wire.FtmInternalMsg //outgoing message queue for factom application messages

	inCtlMsgQueue  chan wire.FtmInternalMsg //incoming message queue for factom control messages
	outCtlMsgQueue chan wire.FtmInternalMsg //outgoing message queue for factom control messages	

	creditsPerChain   int32  = 10
	creditsPerFactoid uint64 = 1000

	// To be moved to ftmMemPool??
	chainIDMap     map[string]*common.EChain // ChainIDMap with chainID string([32]byte) as key
	commitChainMap = make(map[string]*common.CommitChain, 0)
	commitEntryMap = make(map[string]*common.CommitEntry, 0)
	eCreditMap     map[string]int32 // eCreditMap with public key string([32]byte) as key, credit balance as value

	chainIDMapBackup map[string]*common.EChain //previous block bakcup - ChainIDMap with chainID string([32]byte) as key
	eCreditMapBackup map[string]int32          // backup from previous block - eCreditMap with public key string([32]byte) as key, credit balance as value

	//Diretory Block meta data map
	//dbInfoMap map[string]*common.DBInfo // dbInfoMap with dbHash string([32]byte) as key

	fMemPool *ftmMemPool
	plMgr    *consensus.ProcessListMgr

	//Server Private key and Public key for milestone 1
	serverPrivKey common.PrivateKey
	serverPubKey  common.PublicKey

	FactoshisPerCredit uint64 // .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents

	factomdUser string
	factomdPass string
)

var (
	directoryBlockInSeconds int
	dataStorePath           string
	ldbpath                 string
	nodeMode                string
	devNet                  bool
	serverPrivKeyHex        string
)

// Get the configurations 
func LoadConfigurations(cfg *util.FactomdConfig) {
	util.Trace()

	//setting the variables by the valued form the config file
	logLevel = cfg.Log.LogLevel
	dataStorePath = cfg.App.DataStorePath
	ldbpath = cfg.App.LdbPath
	directoryBlockInSeconds = cfg.App.DirectoryBlockInSeconds
	nodeMode = cfg.App.NodeMode
	serverPrivKeyHex = cfg.App.ServerPrivKey

	factomdUser = cfg.Btc.RpcUser
	factomdPass = cfg.Btc.RpcPass
}

// Initialize the processor
func initProcessor() {

	wire.Init()

	util.Trace()

	// init server private key or pub key
	initServerKeys()

	// init mem pools
	fMemPool = new(ftmMemPool)
	fMemPool.init_ftmMemPool()

	// init wire.FChainID
	wire.FChainID = new(common.Hash)
	wire.FChainID.SetBytes(common.FACTOID_CHAINID)

	FactoshisPerCredit = 666667 // .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents

	// init Directory Block Chain
	initDChain()
	fmt.Println("Loaded", dchain.NextBlockHeight, "Directory blocks for chain: "+dchain.ChainID.String())

	// init Entry Credit Chain
	initECChain()
	fmt.Println("Loaded", ecchain.NextBlockHeight, "Entry Credit blocks for chain: "+ecchain.ChainID.String())

	// init Admin Chain
	initAChain()
	fmt.Println("Loaded", achain.NextBlockHeight, "Admin blocks for chain: "+achain.ChainID.String())

	initFctChain()
    common.FactoidState.LoadState()
	fmt.Println("Loaded", scchain.NextBlockHeight, "factoid blocks for chain: "+scchain.ChainID.String())

	anchor.InitAnchor(db)

	// build the Genesis blocks if the current height is 0
	if dchain.NextBlockHeight == 0 {
		buildGenesisBlocks()
	} else {
		/*
			// still send a message to the btcd-side to start up the database; such as a current block height
			eomMsg := &wire.MsgInt_EOM{
				EOM_Type:         wire.INFO_CURRENT_HEIGHT,
				NextDBlockHeight: dchain.NextBlockHeight,
			}
			outCtlMsgQueue <- eomMsg
		*/

		// To be improved in milestone 2
		SignDirectoryBlock()
	}

	// init process list manager
	initProcessListMgr()

	// init Entry Chains
	initEChains()
	for _, chain := range chainIDMap {
		initEChainFromDB(chain)

		fmt.Println("Loaded", chain.NextBlockHeight, "blocks for chain: "+chain.ChainID.String())
		//fmt.Printf("PROCESSOR: echain=%s\n", spew.Sdump(chain))
	}

	// Validate all dir blocks
	err := validateDChain(dchain)
	if err != nil {
		if nodeMode == common.SERVER_NODE {
			panic("Error found in validating directory blocks: " + err.Error())
		} else {
			dchain.IsValidated = false
		}
	}
}

// Started from factomd
func Start_Processor(
	ldb database.Db,
	inMsgQ chan wire.FtmInternalMsg,
	outMsgQ chan wire.FtmInternalMsg,
	inCtlMsgQ chan wire.FtmInternalMsg,
	outCtlMsgQ chan wire.FtmInternalMsg) {
	db = ldb

	inMsgQueue = inMsgQ
	outMsgQueue = outMsgQ

	inCtlMsgQueue = inCtlMsgQ
	outCtlMsgQueue = outCtlMsgQ

	initProcessor()

	// Initialize timer for the open dblock before processing messages
	if nodeMode == common.SERVER_NODE {
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextBlockHeight,
			inCtlMsgQueue:    inCtlMsgQueue,
		}
		go timer.StartBlockTimer()
	} else {
		// start the go routine to process the blocks and entries downloaded from peers
		go validateAndStoreBlocks(fMemPool, db, dchain, outCtlMsgQueue)
	}

	// Process msg from the incoming queue one by one
	for {
		select {
		case msg := <-inMsgQ:
			fmt.Printf("PROCESSOR: in inMsgQ, msg:%+v\n", msg)

			if err := serveMsgRequest(msg); err != nil {
				log.Println(err)
			}

		case ctlMsg := <-inCtlMsgQueue:
			fmt.Printf("PROCESSOR: in ctlMsg, msg:%+v\n", ctlMsg)

			if err := serveMsgRequest(ctlMsg); err != nil {
				log.Println(err)
			}
		}

	}

	util.Trace()

}



// Serve the "fast lane" incoming control msg from inCtlMsgQueue
func serveCtlMsgRequest(msg wire.FtmInternalMsg) error {

	util.Trace()

	switch msg.Command() {
	case wire.CmdCommitChain:

	default:
		return errors.New("Message type unsupported:" + fmt.Sprintf("%+v", msg))
	}
	return nil

}

// Serve incoming msg from inMsgQueue
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
		// Broadcast the msg to the network if no errors
		outMsgQueue <- msg

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
		// Broadcast the msg to the network if no errors
		outMsgQueue <- msg

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
		// Broadcast the msg to the network if no errors
		outMsgQueue <- msg

	case wire.CmdInt_EOM:
		util.Trace("CmdInt_EOM")

		if nodeMode == common.SERVER_NODE {
			msgEom, ok := msg.(*wire.MsgInt_EOM)
			if !ok {
				return errors.New("Error in build blocks:" + fmt.Sprintf("%+v", msg))
			}
			fmt.Printf("PROCESSOR: End of minute msg - wire.CmdInt_EOM:%+v\n", msg)

			if msgEom.EOM_Type == wire.END_MINUTE_10 {
				// Process from Orphan pool before the end of process list
				processFromOrphanPool()

				// Pass the Entry Credit Exchange Rate into the Factoid component
				msgEom.EC_Exchange_Rate = FactoshisPerCredit
				plMgr.AddMyProcessListItem(msgEom, nil, wire.END_MINUTE_10)

				err := buildBlocks()
				if err != nil {
					return err
				}

			} else if msgEom.EOM_Type >= wire.END_MINUTE_1 && msgEom.EOM_Type < wire.END_MINUTE_10 {
				plMgr.AddMyProcessListItem(msgEom, nil, msgEom.EOM_Type)
			}
		}

	case wire.CmdInt_FactoidBlock: // to be removed??
		factoidBlock, ok := msg.(*wire.MsgInt_FactoidBlock)
		util.Trace("Factoid Block (GENERATED??) -- detected in the processor")
		fmt.Println("factoidBlock= ", factoidBlock, " ok= ", ok)

	case wire.CmdDirBlock:
		if nodeMode == common.SERVER_NODE {
			break
		}

		dirBlock, ok := msg.(*wire.MsgDirBlock)
		if ok {
			err := processDirBlock(dirBlock)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdFBlock:
		if nodeMode == common.SERVER_NODE {
			break
		}

		fblock, ok := msg.(*wire.MsgFBlock)
		if ok {
			err := processFBlock(fblock)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdABlock:
		if nodeMode == common.SERVER_NODE {
			break
		}

		ablock, ok := msg.(*wire.MsgABlock)
		if ok {
			err := processABlock(ablock)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdECBlock:
		if nodeMode == common.SERVER_NODE {
			break
		}

		cblock, ok := msg.(*wire.MsgECBlock)
		if ok {
			err := procesECBlock(cblock)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdEBlock:
		if nodeMode == common.SERVER_NODE {
			break
		}

		eblock, ok := msg.(*wire.MsgEBlock)
		if ok {
			err := processEBlock(eblock)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	case wire.CmdTestCredit:
		cred, ok := msg.(*wire.MsgTestCredit)
		if !ok {
			return fmt.Errorf("Error adding test entry credits")
		}
		if err := processTestCredit(cred); err != nil {
			return err
		}

	case wire.CmdEntry:
		if nodeMode == common.SERVER_NODE {
			break
		}

		entry, ok := msg.(*wire.MsgEntry)
		if ok {
			err := processEntry(entry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}

	default:
		return errors.New("Message type unsupported:" + fmt.Sprintf("%+v", msg))
	}

	return nil
}

// processTestCedits assignes credits to a specified publick key for testing
// against the local node. This credit purchase should never propigate across
// the network.
// TODO remove this before production
func processTestCredit(msg *wire.MsgTestCredit) error {
	if _, exists := eCreditMap[string(msg.ECKey[:])]; !exists {
		eCreditMap[string(msg.ECKey[:])] = 0
	}
	eCreditMap[string(msg.ECKey[:])] += msg.Amt
	return nil
}


// processAcknowledgement validates the ack and adds it to processlist
func processAcknowledgement(msg *wire.MsgAcknowledgement) error {
	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	// Validate the signiture
	// To be added ??

	// Update the next block height in dchain
	if msg.Height > dchain.NextBlockHeight {
		dchain.NextBlockHeight = msg.Height
	}

	return nil
}

/* this should be processed on btcd side
// processFactoidBlock validates factoid block and save it to factom db.
func processFactoidBlock(msg *wire.MsgBlock) error {
	util.Trace()
	fmt.Printf("PROCESSOR: MsgFactoidBlock=%s\n", spew.Sdump(msg))
	return nil
}
*/

/*
// Process a factoid obj message and put it in the process list
func processFactoidTx(msg *wire.MsgInt_FactoidObj) error {

	// Update the credit balance in memory for each EC output
	for k, v := range msg.EntryCredits {
		pubKey := new([32]byte)
		copy(pubKey[:], k.Bytes())
		//credits := int32(creditsPerFactoid * v / 100000000)
		// Update the credit balance in memory
		balance, _ := eCreditMap[string(pubKey[:])]
		eCreditMap[string(pubKey[:])] = balance + int32(v)
	}

	// Add to MyPL if Server Node
	if nodeMode == common.SERVER_NODE {
		err := plMgr.AddMyProcessListItem(msg, msg.TxSha, wire.ACK_FACTOID_TX)
		if err != nil {
			return err
		}

	}

	return nil
}
*/

func processRevealEntry(msg *wire.MsgRevealEntry) error {
	e := msg.Entry
	bin, _ := e.MarshalBinary()
	h, _ := wire.NewShaHash(e.Hash().Bytes())

	if c, ok := commitEntryMap[e.Hash().String()]; ok {
		if chainIDMap[e.ChainID.String()] == nil {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("This chain is not supported: %s",
				msg.Entry.ChainID.String())
		}

		cred := int32(binary.Size(bin)/1024 + 1)
		if int32(c.Credits) < cred {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("Credit needs to paid first before an entry is revealed: %s", e.Hash().String())
			// Add the msg to the Mem pool
			fMemPool.addMsg(msg, h)

			// Add to MyPL if Server Node
			if nodeMode == common.SERVER_NODE {
				ack, err := plMgr.AddMyProcessListItem(msg, h, wire.ACK_REVEAL_ENTRY)
				if err != nil {
					return err
				} else {
					// Broadcast the ack to the network if no errors
					outMsgQueue <- ack
				}
			}
		}

		delete(commitEntryMap, e.Hash().String())
		return nil
	} else if c, ok := commitChainMap[e.Hash().String()]; ok {
		if chainIDMap[e.ChainID.String()] != nil {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("This chain is not supported: %s",
				msg.Entry.ChainID.String())
		}

		// add new chain to chainIDMap
		newChain := new(common.EChain)
		newChain.ChainID = e.ChainID
		newChain.FirstEntry = e
		chainIDMap[e.ChainID.String()] = newChain

		cred := int32(binary.Size(bin)/1024 + 1 + 10)
		if int32(c.Credits) < cred {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("Credit needs to paid first before an entry is revealed: %s", e.Hash().String())
			// Add the msg to the Mem pool
			fMemPool.addMsg(msg, h)

			// Add to MyPL if Server Node
			if nodeMode == common.SERVER_NODE {
				ack, err := plMgr.AddMyProcessListItem(msg, h, wire.ACK_REVEAL_ENTRY)
				if err != nil {
					return err
				} else {
					// Broadcast the ack to the network if no errors
					outMsgQueue <- ack
				}
			}
		}

		delete(commitChainMap, e.Hash().String())
		return nil
	} else {
		return fmt.Errorf("No commit for entry")
	}

	return nil
}

func processCommitEntry(msg *wire.MsgCommitEntry) error {
	c := msg.CommitEntry

	// check that the CommitChain is fresh
	if !c.InTime() {
		return fmt.Errorf("Cannot commit chain, CommitChain must be timestamped within 24 hours of commit")
	}

	// check to see if the EntryHash has already been committed
	if _, exist := commitEntryMap[c.EntryHash.String()]; exist {
		return fmt.Errorf("Cannot commit entry, entry has already been commited")
	}

	// add to the commitEntryMap
	commitEntryMap[c.EntryHash.String()] = c

	// Server: add to MyPL
	if nodeMode == common.SERVER_NODE {
		h, _ := msg.Sha()
		ack, err := plMgr.AddMyProcessListItem(msg, &h, wire.ACK_COMMIT_ENTRY)
		if err != nil {
			return err
		} else {
			// Broadcast the ack to the network if no errors
			outMsgQueue <- ack
		}
	}

	return nil
}

func processCommitChain(msg *wire.MsgCommitChain) error {
	c := msg.CommitChain

	// check that the CommitChain is fresh
	if !c.InTime() {
		return fmt.Errorf("Cannot commit chain, CommitChain must be timestamped within 24 hours of commit")
	}

	// check to see if the EntryHash has already been committed
	if _, exist := commitChainMap[c.EntryHash.String()]; exist {
		return fmt.Errorf("Cannot commit chain, first entry for chain already exists")
	}

	// deduct the entry credits from the eCreditMap
	if eCreditMap[string(c.ECPubKey[:])] < int32(c.Credits) {
		return fmt.Errorf("Not enough credits for CommitChain")
	}
	eCreditMap[string(c.ECPubKey[:])] -= int32(c.Credits)

	// add to the commitChainMap
	commitChainMap[c.EntryHash.String()] = c

	// Server: add to MyPL
	if nodeMode == common.SERVER_NODE {
		h, _ := msg.Sha()
		ack, err := plMgr.AddMyProcessListItem(msg, &h, wire.ACK_COMMIT_CHAIN)
		if err != nil {
			return err
		} else {
			// Broadcast the ack to the network if no errors
			outMsgQueue <- ack
		}
	}

	return nil
}

func processBuyEntryCredit(pubKey *[32]byte, credits int32, factoidTxHash *common.Hash) error {

	// Update the credit balance in memory
	balance, _ := eCreditMap[string(pubKey[:])]
	eCreditMap[string(pubKey[:])] = balance + credits

	return nil
}

// Process Orphan pool before the end of 10 min
func processFromOrphanPool() error {
	for k, msg := range fMemPool.orphans {
		switch msg.Command() {
		case wire.CmdCommitChain:
			msgCommitChain, _ := msg.(*wire.MsgCommitChain)
			err := processCommitChain(msgCommitChain)
			if err != nil {
				return err
			}
			delete(fMemPool.orphans, k)

		case wire.CmdCommitEntry:
			msgCommitEntry, _ := msg.(*wire.MsgCommitEntry)
			err := processCommitEntry(msgCommitEntry)
			if err != nil {
				return err
			}
			delete(fMemPool.orphans, k)

		case wire.CmdRevealEntry:
			msgRevealEntry, _ := msg.(*wire.MsgRevealEntry)
			err := processRevealEntry(msgRevealEntry)
			if err != nil {
				return err
			}
			delete(fMemPool.orphans, k)
		}
	}
	return nil
}

func buildRevealEntry(msg *wire.MsgRevealEntry) {

	chain := chainIDMap[msg.Entry.ChainID.String()]

	// store the new entry in db
	entryBinary, _ := msg.Entry.MarshalBinary()
	entryHash := common.Sha(entryBinary)

	db.InsertEntry(entryHash, msg.Entry)

	err := chain.NextBlock.AddEBEntry(msg.Entry)

	if err != nil {
		panic("Error while adding Entity to Block:" + err.Error())
	}

}

func buildCommitEntry(msg *wire.MsgCommitEntry) {
	ecchain.NextBlock.AddEntry(msg.CommitEntry)
}

func buildCommitChain(msg *wire.MsgCommitChain) {
	ecchain.NextBlock.AddEntry(msg.CommitChain)
}

/*
func buildFactoidObj(msg *wire.MsgInt_FactoidObj) {
	factoidTxHash := new(common.Hash)
	factoidTxHash.SetBytes(msg.TxSha.Bytes())

	for k, v := range msg.EntryCredits {
		pubkey := new([32]byte)
		copy(pubkey[:], k.Bytes())
		cbEntry := common.NewIncreaseBalance(pubkey, factoidTxHash, int32(v))
		ecchain.NextBlock.AddEntry(cbEntry)
	}
}
*/

func buildRevealChain(msg *wire.MsgRevealChain) {

	newChain := chainIDMap[msg.FirstEntry.ChainID.String()]

	// Store the new chain in db
	db.InsertChain(newChain)

	// Chain initialization
	initEChainFromDB(newChain)

	// store the new entry in db
	entryBinary, _ := newChain.FirstEntry.MarshalBinary()
	entryHash := common.Sha(entryBinary)
	db.InsertEntry(entryHash, newChain.FirstEntry)

	err := newChain.NextBlock.AddEBEntry(newChain.FirstEntry)

	if err != nil {
		panic(fmt.Sprintf(`Error while adding the First Entry to Block: %s`, err.Error()))
	}
}

// Loop through the Process List items and get the touched chains
// Put End-Of-Minute marker in the entry chains
func buildEndOfMinute(pl *consensus.ProcessList, pli *consensus.ProcessListItem) {
	tempChainMap := make(map[string]*common.EChain)
	items := pl.GetPLItems()
	for i := pli.Ack.Index; i >= 0; i-- {
		if wire.END_MINUTE_1 <= items[i].Ack.Type && items[i].Ack.Type <= wire.END_MINUTE_10 {
			break
		} else if items[i].Ack.Type == wire.ACK_REVEAL_ENTRY && tempChainMap[items[i].Ack.ChainID.String()] == nil {

			chain := chainIDMap[items[i].Ack.ChainID.String()]
			chain.NextBlock.AddEndOfMinuteMarker(pli.Ack.Type)
			// Add the new chain in the tempChainMap
			tempChainMap[chain.ChainID.String()] = chain
		}
	}

	// Add it to the entry credit chain
	entries := ecchain.NextBlock.Body.Entries
	if len(entries) > 0 && entries[len(entries)-1].ECID() != common.ECIDMinuteNumber {
		cbEntry := common.NewMinuteNumber()
		cbEntry.Number = pli.Ack.Type
		ecchain.NextBlock.AddEntry(cbEntry)
	}

	// Add it to the admin chain
	abEntries := achain.NextBlock.ABEntries
	if len(abEntries) > 0 && abEntries[len(abEntries)-1].Type() != common.TYPE_MINUTE_NUM {
		achain.NextBlock.AddEndOfMinuteMarker(pli.Ack.Type)
	}
}

// build Genesis blocks
func buildGenesisBlocks() error {

	/*
		// Send an End of Minute message to the Factoid component to create a genesis block
		eomMsg := &wire.MsgInt_EOM{
			EOM_Type:         wire.FORCE_FACTOID_GENESIS_REBUILD,
			NextDBlockHeight: 0,
		}
		outCtlMsgQueue <- eomMsg
	*/

	// Allocate the first two dbentries for ECBlock and Factoid block
	dchain.AddDBEntry(&common.DBEntry{}) // AdminBlock
	dchain.AddDBEntry(&common.DBEntry{}) // ECBlock
	dchain.AddDBEntry(&common.DBEntry{}) // Factoid block

	// Entry Credit Chain
	cBlock := newEntryCreditBlock(ecchain)
	fmt.Printf("buildGenesisBlocks: cBlock=%s\n", spew.Sdump(cBlock))
	dchain.AddECBlockToDBEntry(cBlock)
	exportECChain(ecchain)

	// Admin chain
	aBlock := newAdminBlock(achain)
	fmt.Printf("buildGenesisBlocks: aBlock=%s\n", spew.Sdump(aBlock))
	dchain.AddABlockToDBEntry(aBlock)
	exportAChain(achain)

	// factoid Genesis Address
	FBlock := newFactoidBlock(scchain)
	data, _ := FBlock.MarshalBinary()
	fmt.Println("\n\n ", common.Sha(data).String(), "\n\n")
	dchain.AddFBlockToDBEntry(FBlock)
	exportFctChain(scchain)

	// Directory Block chain
	util.Trace("in buildGenesisBlocks")
	dbBlock := newDirectoryBlock(dchain)

	// Check block hash if genesis block
	if dbBlock.DBHash.String() != common.GENESIS_DIR_BLOCK_HASH {

		panic("\nGenesis block hash expected: " + common.GENESIS_DIR_BLOCK_HASH +
			"\nGenesis block hash found:    " + dbBlock.DBHash.String() + "\n")
	}

	exportDChain(dchain)

	// place an anchor into btc
	placeAnchor(dbBlock)

	return nil
}

// build blocks from all process lists
func buildBlocks() error {
	util.Trace()

	// Allocate the first three dbentries for Admin block, ECBlock and Factoid block
	dchain.AddDBEntry(&common.DBEntry{}) // AdminBlock
	dchain.AddDBEntry(&common.DBEntry{}) // ECBlock
	dchain.AddDBEntry(&common.DBEntry{}) // factoid

	if plMgr != nil && plMgr.MyProcessList.IsValid() {
		buildFromProcessList(plMgr.MyProcessList)
	}

	// Entry Credit Chain
	ecBlock := newEntryCreditBlock(ecchain)
	dchain.AddECBlockToDBEntry(ecBlock)
	exportECChain(ecchain)

	// Admin chain
	aBlock := newAdminBlock(achain)
	//fmt.Printf("buildGenesisBlocks: aBlock=%s\n", spew.Sdump(aBlock))
	dchain.AddABlockToDBEntry(aBlock)
	exportAChain(achain)

	// Factoid chain
	fBlock := newFactoidBlock(scchain)
	//fmt.Printf("buildGenesisBlocks: aBlock=%s\n", spew.Sdump(aBlock))
	dchain.AddFBlockToDBEntry(fBlock)
	exportFctChain(scchain)

	// sort the echains by chain id
	var keys []string
	for k := range chainIDMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Entry Chains
	for _, k := range keys {
		chain := chainIDMap[k]
		eblock := newEntryBlock(chain)
		if eblock != nil {
			dchain.AddEBlockToDBEntry(eblock)
		}
		exportEChain(chain)
	}

	// Directory Block chain
	util.Trace("in buildBlocks")
	dbBlock := newDirectoryBlock(dchain)
	// Check block hash if genesis block here??

	// Generate the inventory vector and relay it.
	binary, _ := dbBlock.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())
	outMsgQueue <- (&wire.MsgInt_DirBlock{hash})
	
	// Update dir block height cache in db
	db.UpdateBlockHeightCache(dbBlock.Header.BlockHeight, commonHash)	

	exportDChain(dchain)

	// re-initialize the process lit manager
	initProcessListMgr()

	// Initialize timer for the new dblock
	if nodeMode == common.SERVER_NODE {
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextBlockHeight,
			inCtlMsgQueue:    inCtlMsgQueue,
		}
		go timer.StartBlockTimer()
	}

	// place an anchor into btc
	placeAnchor(dbBlock)

	return nil
}


// build blocks from a process lists
func buildFromProcessList(pl *consensus.ProcessList) error {
	for _, pli := range pl.GetPLItems() {
		if pli.Ack.Type == wire.ACK_COMMIT_CHAIN {
			buildCommitChain(pli.Msg.(*wire.MsgCommitChain))
		} else if pli.Ack.Type == wire.ACK_COMMIT_ENTRY {
			buildCommitEntry(pli.Msg.(*wire.MsgCommitEntry))
		} else if pli.Ack.Type == wire.ACK_REVEAL_CHAIN {
			buildRevealChain(pli.Msg.(*wire.MsgRevealChain))
		} else if pli.Ack.Type == wire.ACK_REVEAL_ENTRY {
			buildRevealEntry(pli.Msg.(*wire.MsgRevealEntry))
		} else if wire.END_MINUTE_1 <= pli.Ack.Type && pli.Ack.Type <= wire.END_MINUTE_10 {
			buildEndOfMinute(pl, pli)
		}
	}

	return nil
}

func newEntryBlock(chain *common.EChain) *common.EBlock {

	// acquire the last block
	block := chain.NextBlock

	if len(block.EBEntries) < 1 {
		//log.Println("No new entry found. No block created for chain: "  + common.EncodeChainID(chain.ChainID))
		return nil
	}

	// Create the block and add a new block for new coming entries

	block.Header.DBHeight = dchain.NextBlockHeight
	block.Header.EntryCount = uint32(len(block.EBEntries))
	block.Header.StartTime = dchain.NextBlock.Header.StartTime

	if devNet {
		block.Header.NetworkID = common.NETWORK_ID_TEST
	} else {
		block.Header.NetworkID = common.NETWORK_ID_EB
	}

	// Create the Entry Block Boday Merkle Root from EB Entries
	hashes := make([]*common.Hash, 0, len(block.EBEntries))
	for _, entry := range block.EBEntries {
		hashes = append(hashes, entry.EntryHash)
	}
	merkle := common.BuildMerkleTreeStore(hashes)
	block.Header.BodyMR = merkle[len(merkle)-1]

	// Create the Entry Block Key Merkle Root from the hash of Header and the Body Merkle Root
	hashes = make([]*common.Hash, 0, 2)
	binaryEBHeader, _ := block.Header.MarshalBinary()
	hashes = append(hashes, common.Sha(binaryEBHeader))
	hashes = append(hashes, block.Header.BodyMR)
	merkle = common.BuildMerkleTreeStore(hashes)
	block.MerkleRoot = merkle[len(merkle)-1] // MerkleRoot is not marshalized in Entry Block
	fmt.Println("block.MerkleRoot:%v", block.MerkleRoot.String())
	blkhash, _ := common.CreateHash(block)
	block.EBHash = blkhash
	log.Println("blkhash:%v", blkhash.Bytes())

	block.IsSealed = true
	chain.NextBlockHeight++
	chain.NextBlock, _ = common.CreateBlock(chain, block, 10)

	//Store the block in db
	db.ProcessEBlockBatch(block)
	log.Println("EntryBlock: block" + strconv.FormatUint(uint64(block.Header.EBHeight), 10) + " created for chain: " + chain.ChainID.String())
	return block
}

func newEntryCreditBlock(chain *common.ECChain) *common.ECBlock {

	// acquire the last block
	block := chain.NextBlock

	if chain.NextBlockHeight != dchain.NextBlockHeight {
		panic("Entry Credit Block height does not match Directory Block height:" + string(dchain.NextBlockHeight))
	}

	block.BuildHeader()

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	chain.NextBlock = common.NextECBlock(block)
	chain.BlockMutex.Unlock()

	//Store the block in db
	db.ProcessECBlockBatch(block)
	log.Println("EntryCreditBlock: block" + strconv.FormatUint(uint64(block.Header.DBHeight), 10) + " created for chain: " + chain.ChainID.String())

	return block
}

func newAdminBlock(chain *common.AdminChain) *common.AdminBlock {

	// acquire the last block
	block := chain.NextBlock

	if chain.NextBlockHeight != dchain.NextBlockHeight {
		panic("Admin Block height does not match Directory Block height:" + string(dchain.NextBlockHeight))
	}

	block.Header.EntryCount = uint32(len(block.ABEntries))
	block.Header.BodySize = uint32(block.MarshalledSize() - block.Header.MarshalledSize())
	block.BuildABHash()

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	chain.NextBlock, _ = common.CreateAdminBlock(chain, block, 10)
	chain.BlockMutex.Unlock()

	//Store the block in db
	db.ProcessABlockBatch(block)
	log.Println("Admin Block: block" + strconv.FormatUint(uint64(block.Header.DBHeight), 10) + " created for chain: " + chain.ChainID.String())

	return block
}

func newFactoidBlock(chain *common.FctChain) block.IFBlock {

	// acquire the last block
	currentBlock := chain.NextBlock

	if chain.NextBlockHeight != dchain.NextBlockHeight {
		panic("Factoid Block height does not match Directory Block height:" + strconv.Itoa(int(dchain.NextBlockHeight)))
	}

	//block.BuildHeader()

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	chain.NextBlock = block.NewFBlock(FactoshisPerCredit, chain.NextBlockHeight)
	chain.BlockMutex.Unlock()

	//Store the block in db
	db.ProcessFBlockBatch(currentBlock)
	log.Println("Factoid chain: block" + " created for chain: " + chain.ChainID.String())

	return currentBlock
}

func newDirectoryBlock(chain *common.DChain) *common.DirectoryBlock {
	util.Trace("**** new Dir Block")
	// acquire the last block
	block := chain.NextBlock

	if devNet {
		block.Header.NetworkID = common.NETWORK_ID_TEST
	} else {
		block.Header.NetworkID = common.NETWORK_ID_EB
	}

	// Create the block add a new block for new coming entries
	chain.BlockMutex.Lock()
	block.Header.EntryCount = uint32(len(block.DBEntries))
	// Calculate Merkle Root for FBlock and store it in header
	if block.Header.BodyMR == nil {
		block.Header.BodyMR, _ = block.BuildBodyMR()
		//  Factoid1 block not in the right place...
	}
	block.IsSealed = true
	chain.AddDBlockToDChain(block)
	chain.NextBlockHeight++
	chain.NextBlock, _ = common.CreateDBlock(chain, block, 10)
	chain.BlockMutex.Unlock()

	block.DBHash, _ = common.CreateHash(block)
	block.BuildKeyMerkleRoot()

	//Store the block in db
	db.ProcessDBlockBatch(block)

	// Initialize the dirBlockInfo obj in db
	db.InsertDirBlockInfo(common.NewDirBlockInfoFromDBlock(block))
	anchor.UpdateDirBlockInfoMap(common.NewDirBlockInfoFromDBlock(block))

	log.Println("DirectoryBlock: block" + strconv.FormatUint(uint64(block.Header.BlockHeight), 10) + " created for directory block chain: " + chain.ChainID.String())

	// To be improved in milestone 2
	SignDirectoryBlock()

	return block
}


// Sign the directory block
func SignDirectoryBlock() error {
	// Only Servers can write the anchor to Bitcoin network
	if nodeMode == common.SERVER_NODE && dchain.NextBlockHeight > 0 {
		// get the previous directory block from db
		dbBlock, _ := db.FetchDBlockByHeight(dchain.NextBlockHeight - 1)
		dbHeaderBytes, _ := dbBlock.Header.MarshalBinary()
		identityChainID := common.NewHash() // 0 ID for milestone 1
		sig := serverPrivKey.Sign(dbHeaderBytes)
		achain.NextBlock.AddABEntry(common.NewDBSignatureEntry(identityChainID, sig))
	}

	return nil
}

// Place an anchor into btc
func placeAnchor(dbBlock *common.DirectoryBlock) error {
	util.Trace()
	// Only Servers can write the anchor to Bitcoin network
	if nodeMode == common.SERVER_NODE && dbBlock != nil {
		// todo: need to make anchor as a go routine, independent of factomd
		// same as blockmanager to btcd
		go anchor.SendRawTransactionToBTC(dbBlock.KeyMR, uint64(dbBlock.Header.BlockHeight))
	}
	return nil
}
