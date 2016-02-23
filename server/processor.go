// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.
// github.com/alexcesaro/log/golog (MIT License)

// Processor is the engine of Factom.
// It processes all of the incoming messages from the network.
// It syncs up with peers and build blocks based on the process lists and a
// timed schedule.
// For details, please refer to:
// https://github.com/FactomProject/FactomDocs/blob/master/FactomLedgerbyConsensus.pdf

package server

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wire"
	fct "github.com/FactomProject/factoid"
	"github.com/FactomProject/factoid/block"
	"github.com/davecgh/go-spew/spew"
)

var _ = (*block.FBlock)(nil)

var _ = util.Trace

var (
	localServer  *server
	db           database.Db // database
	factomConfig *util.FactomdConfig
	inMsgQueue   chan wire.FtmInternalMsg
	outMsgQueue  chan wire.FtmInternalMsg

	dchain   *common.DChain     //Directory Block Chain
	ecchain  *common.ECChain    //Entry Credit Chain
	achain   *common.AdminChain //Admin Chain
	fchain   *common.FctChain   // factoid Chain
	fchainID *common.Hash

	newDBlock  *common.DirectoryBlock
	newABlock  *common.AdminBlock
	newFBlock  block.IFBlock
	newECBlock *common.ECBlock
	newEBlocks []*common.EBlock

	//TODO: To be moved to ftmMemPool??
	chainIDMap     map[string]*common.EChain // ChainIDMap with chainID string([32]byte) as key
	commitChainMap = make(map[string]*common.CommitChain, 0)
	commitEntryMap = make(map[string]*common.CommitEntry, 0)
	eCreditMap     map[string]int32 // eCreditMap with public key string([32]byte) as key, credit balance as value

	chainIDMapBackup map[string]*common.EChain //previous block bakcup - ChainIDMap with chainID string([32]byte) as key
	eCreditMapBackup map[string]int32          // backup from previous block - eCreditMap with public key string([32]byte) as key, credit balance as value

	fMemPool *ftmMemPool
	plMgr    *consensus.ProcessListMgr

	//Server Private key and Public key for milestone 1
	serverPrivKey common.PrivateKey
	serverPubKey  common.PublicKey

	// FactoshisPerCredit is .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents
	FactoshisPerCredit uint64
	blockSyncing       bool
	firstBlockHeight   uint32 // the DBHeight of the first block being built by follower after sync up
	//doneSyncChan       = make(chan struct{})
	zeroHash = common.NewHash()

	directoryBlockInSeconds int
	dataStorePath           string
	ldbpath                 string
	nodeMode                string
	devNet                  bool
	serverPrivKeyHex        string
	serverIndex             = common.NewServerIndexNumber() //???
)

// LoadConfigurations gets the configurations
func LoadConfigurations(fcfg *util.FactomdConfig) {
	//setting the variables by the valued form the config file
	factomConfig = fcfg
	dataStorePath = factomConfig.App.DataStorePath
	ldbpath = factomConfig.App.LdbPath
	directoryBlockInSeconds = factomConfig.App.DirectoryBlockInSeconds
	nodeMode = factomConfig.App.NodeMode
	serverPrivKeyHex = factomConfig.App.ServerPrivKey
	cp.CP.SetPort(factomConfig.Controlpanel.Port)
}

// InitProcessor initializes the processor
func InitProcessor(ldb database.Db) {
	db = ldb
	initServerKeys()
	fMemPool = new(ftmMemPool)
	fMemPool.initFtmMemPool()
	FactoshisPerCredit = 666666 // .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents

	// init Directory Block Chain
	initDChain()
	fmt.Println("Loaded ", dchain.NextDBHeight, " Directory blocks for chain: "+dchain.ChainID.String())

	// init Entry Credit Chain
	initECChain()
	fmt.Println("Loaded ", ecchain.NextBlockHeight, " Entry Credit blocks for chain: "+ecchain.ChainID.String())

	// init Admin Chain
	initAChain()
	fmt.Println("Loaded ", achain.NextBlockHeight, " Admin blocks for chain: "+achain.ChainID.String())

	initFctChain()
	fmt.Println("Loaded ", fchain.NextBlockHeight, " factoid blocks for chain: "+fchain.ChainID.String())

	//Init anchor for server
	if nodeMode == common.SERVER_NODE {
		anchor.InitAnchor(db, inMsgQueue, serverPrivKey)
	}

	// init process list manager
	initProcessListMgr()

	// init Entry Chains
	initEChains()
	for _, chain := range chainIDMap {
		initEChainFromDB(chain)

		fmt.Println("Loaded ", chain.NextBlockHeight, " blocks for chain: "+chain.ChainID.String())
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

// StartProcessor is started from factomd
func StartProcessor() {
	//initProcessor()
	fmt.Println("StartProcessor: blockSyncing=", blockSyncing)
	if dchain.NextDBHeight == 0 && localServer.IsLeader() { //nodeMode == common.SERVER_NODE && !blockSyncing {
		buildGenesisBlocks()
		procLog.Debug("after creating genesis block: dchain.NextDBHeight=", dchain.NextDBHeight)
	}

	// Initialize timer for the open dblock before processing messages
	//if nodeMode == common.SERVER_NODE && !blockSyncing {
	if localServer.IsLeader() {
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextDBHeight,
			inMsgQueue:       inMsgQueue,
		}
		go timer.StartBlockTimer()
	} else {
		//if !localServer.IsLeader() {
		// process the blocks and entries downloaded from peers
		// this is needed for clients and followers when sync up
		fmt.Println("StartProcessor: validateAndStoreBlocks")
		go validateAndStoreBlocks(fMemPool, db, dchain)
	}

	// Process msg from the incoming queue one by one
	for {
		select {
		case inmsg := <-inMsgQueue:
			if err := serveMsgRequest(inmsg); err != nil {
				procLog.Error(err)
			}

		case msg := <-outMsgQueue:
			switch msg.(type) {
			case *wire.MsgInt_DirBlock:
				// once it's done block syncing, this is only needed for CLIENT
				// Use broadcast to exclude federate servers
				// todo ???
				dirBlock, _ := msg.(*wire.MsgInt_DirBlock)
				iv := wire.NewInvVect(wire.InvTypeFactomDirBlock, dirBlock.ShaHash)
				localServer.RelayInventory(iv, nil)
				//msgDirBlock := &wire.MsgDirBlock{DBlk: dirBlock}
				//excludedPeers := make([]*peer, 0, 32)
				//for e := localServer.federateServers.Front(); e != nil; e = e.Next() {
				//excludedPeers = append(excludedPeers, e.Value.(*peer))
				//}
				//s.BroadcastMessage(msgDirBlock, excludedPeers)

			case wire.Message:
				// verify if this wireMsg should be one of MsgEOM, MsgAck,
				// commitEntry/chain, revealEntry/Chain and MsgDirBlockSig
				// need to exclude all peers that are not federate servers
				// todo ???
				wireMsg, _ := msg.(wire.Message)
				localServer.BroadcastMessage(wireMsg)
				/*
					if ClientOnly {
						//fmt.Println("broadcasting from client.")
						s.BroadcastMessage(wireMsg)
					} else {
						if _, ok := msg.(*wire.MsgAck); ok {
							//fmt.Println("broadcasting from server.")
							s.BroadcastMessage(wireMsg)
						}
					}*/

			default:
				panic(fmt.Sprintf("bad outMsgQueue message received: %v", msg))
			}
		}
	}
}

// Serve incoming msg from inMsgQueue
func serveMsgRequest(msg wire.FtmInternalMsg) error {
	//procLog.Infof("serveMsgRequest: %s", spew.Sdump(msg))
	switch msg.Command() {
	case wire.CmdCommitChain:
		msgCommitChain, _ := msg.(*wire.MsgCommitChain)
		if msgCommitChain.IsValid() { //&& !blockSyncing {

			h := msgCommitChain.CommitChain.GetSigHash().Bytes()
			t := msgCommitChain.CommitChain.GetMilliTime() / 1000

			if !IsTSValid(h, t) {
				return fmt.Errorf("Timestamp invalid on Commit Chain")
			}

			err := processCommitChain(msgCommitChain)
			if err != nil {
				return err
			}
		}
		// Broadcast the msg to the network if no errors
		outMsgQueue <- msg

	case wire.CmdCommitEntry:
		msgCommitEntry, ok := msg.(*wire.MsgCommitEntry)
		if ok && msgCommitEntry.IsValid() {

			h := msgCommitEntry.CommitEntry.GetSigHash().Bytes()
			t := msgCommitEntry.CommitEntry.GetMilliTime() / 1000

			if !IsTSValid(h, t) {
				return fmt.Errorf("Timestamp invalid on Commit Entry")
			}

			err := processCommitEntry(msgCommitEntry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + spew.Sdump(msg))
		}
		// Broadcast the msg to the network if no errors
		outMsgQueue <- msg

	case wire.CmdRevealEntry:
		msgRevealEntry, ok := msg.(*wire.MsgRevealEntry)
		if ok && msgRevealEntry.IsValid() {
			err := processRevealEntry(msgRevealEntry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + spew.Sdump(msg))
		}
		// Broadcast the msg to the network if no errors
		outMsgQueue <- msg

	case wire.CmdAck:
		// only post-syncup followers need to deal with Ack
		if nodeMode != common.SERVER_NODE || localServer.IsLeader() {
			return nil
		}
		ack, _ := msg.(*wire.MsgAck)
		_, latestHeight, _ := db.FetchBlockHeightCache()
		fmt.Printf("in case.CmdAck:: Ack.Height=%d, dchain.NextDBHeight=%d, db.latestDBHeight=%d, blockSyncing=%v\n",
			ack.Height, dchain.NextDBHeight, latestHeight, blockSyncing)
		//dchain.NextDBHeight is the dir block height for the network
		//update it with ack height from the leader if necessary
		if dchain.NextDBHeight < ack.Height {
			dchain.NextDBHeight = ack.Height
		}
		//switch from block syncup to block build
		//if dchain.NextDBHeight == db.FetchBlockHeightCache()+1 {	//&& blockSyncing {
		if IsDChainInSync() && blockSyncing {
			blockSyncing = false
			firstBlockHeight = ack.Height
			// set this federate server's FirstJoined = firstBlockHeight
			fmt.Println("** reset blockSyncing=false, firstBlockHeight=", firstBlockHeight)
		}
		if blockSyncing {
			return nil
		}

		//msgEom, _ := msg.(*wire.MsgInt_EOM)
		//var singleServerMode = localServer.isSingleServerMode()
		//fmt.Println("number of federate servers: ", localServer.FederateServerCount(),
		//"singleServerMode=", singleServerMode)

		// to simplify this, for leader & followers, use the next wire.END_MINUTE_1
		// to trigger signature comparison of last round.
		// todo: when to start? can NOT do this for the first EOM_1 ???
		if ack.Type == wire.END_MINUTE_1 {
			// need to bypass the first block of newly-joined follower, if
			// this is 11th minute: ack.NextDBlockHeight-1 == firstBlockHeight
			// or is 1st minute: fMemPool.LenDirBlockSig() == 0
			fmt.Println("bypass save this block?? firstBlockHeight=", firstBlockHeight, ", ack=", spew.Sdump(ack))
			if fMemPool.LenDirBlockSig() > 0 { //&& ack.Height-1 != firstBlockHeight { //!singleServerMode &&
				go processDirBlockSig()
			} else {
				// three cases go in here
				// a. newDBlock is nil: first EOM_1 with no sync up needed for this follower
				// b. newDBlock is not nil but usually wrong with missing msg or ack: first EOM_1 after sync up done for this follower. need to bypass save but need sync up ???
				// c. newDBlock is the genesis block or normal single server mode
				if newDBlock != nil { //&& ack.Height-1 != firstBlockHeight {
					go saveBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)
				}
			}
		}
		err := processAck(ack)
		if err != nil {
			return err
		}

	case wire.CmdDirBlockSig:
		//only when server is building blocks. relax it for now ???
		// for client, do nothing; for leader, always add it
		// for followers, only add it when done with sync up.
		if nodeMode != common.SERVER_NODE { //|| blockSyncing {
			return nil
		}
		dbs, _ := msg.(*wire.MsgDirBlockSig)
		fmt.Printf("Incoming MsgDirBlockSig: %s\n", spew.Sdump(dbs))
		// to simplify this, use the next wire.END_MINUTE_1 to trigger signature comparison. ???
		fMemPool.addDirBlockSig(dbs)

	case wire.CmdInt_EOM:
		// this is only for leader. as followers have no blocktimer
		// followers EOM will be driven by Ack of this EOM.
		fmt.Printf("in case.CmdInt_EOM: localServer.IsLeader=%v\n", localServer.IsLeader())
		msgEom, _ := msg.(*wire.MsgInt_EOM)
		var singleServerMode = localServer.isSingleServerMode()
		fmt.Println("number of federate servers: ", localServer.FederateServerCount(),
			", singleServerMode=", singleServerMode)

		// todo: need to sync up server & peer, and make sure it connects to
		// at least one peer if available before server getting here.
		// this is mostly a problem of 60 seconds block rather than normally 10 minute block
		// For now, single server mode has to have InitStart = false in config
		// ???
		//if !localServer.IsLeader() {
		//return nil
		//}

		// to simplify this, for leader & followers, use the next wire.END_MINUTE_1
		// to trigger signature comparison of last round.
		// todo: when to start? can NOT do this for the first EOM_1 ???
		if msgEom.EOM_Type == wire.END_MINUTE_1 {
			// need to bypass the first block of newly-joined follower
			// this is 11th minute.
			//fmt.Println("bypass save this block?? firstBlockHeight=", firstBlockHeight, ", msgEom=", spew.Sdump(msgEom))
			if !singleServerMode && fMemPool.LenDirBlockSig() > 0 { //msgEom.NextDBlockHeight-1 != firstBlockHeight {
				go processDirBlockSig()
			} else {
				if newDBlock != nil { //&& msgEom.NextDBlockHeight-1 != firstBlockHeight {
					go saveBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)
				}
			}
		}
		err := processLeaderEOM(msgEom)
		if err != nil {
			return err
		}
		cp.CP.AddUpdate(
			"MinMark",  // tag
			"status",   // Category
			"Progress", // Title
			fmt.Sprintf("End of Minute %v\n", msgEom.EOM_Type)+ // Message
				fmt.Sprintf("Directory Block Height %v", dchain.NextDBHeight),
			0)

	case wire.CmdDirBlock:
		fmt.Printf("wire.CmdDirBlock: blockSyncing: %t\n", blockSyncing)
		if nodeMode == common.SERVER_NODE && !blockSyncing {
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

		if nodeMode == common.SERVER_NODE && !blockSyncing {
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

	case wire.CmdFactoidTX:

		// First check that the message is good, and is valid.  If not,
		// continue processing commands.
		msgFactoidTX, ok := msg.(*wire.MsgFactoidTX)
		if !ok || !msgFactoidTX.IsValid() {
			break
		}
		// prevent replay attacks
		//{
		h := msgFactoidTX.Transaction.GetSigHash().Bytes()
		t := int64(msgFactoidTX.Transaction.GetMilliTimestamp() / 1000)

		if !IsTSValid(h, t) {
			return fmt.Errorf("Timestamp invalid on Factoid Transaction")
		}
		//}

		// Handle the server case
		if nodeMode == common.SERVER_NODE && !blockSyncing {
			t := msgFactoidTX.Transaction
			txnum := len(common.FactoidState.GetCurrentBlock().GetTransactions())
			if common.FactoidState.AddTransaction(txnum, t) == nil {
				if err := processBuyEntryCredit(msgFactoidTX); err != nil {
					return err
				}
			}
		} else {
			// Handle the client case
			outMsgQueue <- msg
		}

	case wire.CmdABlock:
		if nodeMode == common.SERVER_NODE && !blockSyncing {
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
		if nodeMode == common.SERVER_NODE && !blockSyncing {
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
		if nodeMode == common.SERVER_NODE && !blockSyncing {
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

	case wire.CmdEntry:
		if nodeMode == common.SERVER_NODE && !blockSyncing {
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

func processLeaderEOM(msgEom *wire.MsgInt_EOM) error {
	//procLog.Infof("processLeaderEOM: wire.CmdInt_EOM:%+v\n", msgEom)
	common.FactoidState.EndOfPeriod(int(msgEom.EOM_Type))
	if msgEom.EOM_Type == wire.END_MINUTE_10 {
		// Process from Orphan pool before the end of process list
		fmt.Println("processLeaderEOM: END_MINUTE_10: before processFromOrphanPool")
		processFromOrphanPool()
	}

	ack, err := plMgr.AddToLeadersProcessList(msgEom, nil, msgEom.EOM_Type)
	if err != nil {
		return err
	}
	//???
	if ack.ChainID == nil {
		ack.ChainID = dchain.ChainID
	}
	fmt.Printf("processLeaderEOM: leader sending ack=%s\n", spew.Sdump(ack))
	outMsgQueue <- ack

	//procLog.Infof("current ProcessList: %s", spew.Sdump(plMgr.MyProcessList))
	if msgEom.EOM_Type == wire.END_MINUTE_10 {
		fmt.Println("processLeaderEOM: END_MINUTE_10: before LEADER buildBlocks")
		err = buildBlocks() //broadcast new dir block sig
		if err != nil {
			return err
		}
	}
	return nil
}

// processDirBlockSig check received MsgDirBlockMsg in mempool and
// decide which dir block to save to database and for anchor.
func processDirBlockSig() error {
	dbsigs := fMemPool.getDirBlockSigPool()
	if len(dbsigs) == 0 {
		fmt.Println("no dir block sig in mempool.")
		return nil
	}
	totalServerNum := localServer.FederateServerCount()
	fmt.Printf("processDirBlockSig(): By EOM_1, there're %d dirblock signatures arrived out of %d federate servers.\n",
		len(dbsigs), totalServerNum)
	//fmt.Println("processDirBlockSig(): DirBlockSigPool: ", spew.Sdump(dbsigs))

	dgsMap := make(map[string][]*wire.MsgDirBlockSig)
	for _, v := range dbsigs {
		//if !v.Sig.Pub.Verify(v.DirBlockHash.Bytes(), v.Sig.Sig) {
		//fmt.Println("could not verify sig. dir block hash: ", v.DirBlockHash)
		//continue
		//}
		if v.DBHeight != dchain.NextDBHeight-1 {
			// need to remove this oen
			fmt.Println("filter out later-coming last block's sig: ", spew.Sdump(v))
			continue
		}
		key := v.DirBlockHash.String()
		val := dgsMap[key]
		//fmt.Printf("key0=%s, dir block sig=%s\n", key, spew.Sdump(val))
		if val == nil {
			//fmt.Println("sig is nil.")
			val = make([]*wire.MsgDirBlockSig, 0, 32)
			val = append(val, v)
			dgsMap[key] = val
		} else {
			val = append(val, v)
			dgsMap[key] = val
		}
		//fmt.Printf("key=%s, dir block sig=%s\n", key, spew.Sdump(dgsMap[key]))
	}

	var winner *wire.MsgDirBlockSig
	for _, v := range dgsMap {
		n := float32(len(v)) / float32(totalServerNum)
		//fmt.Printf("key=%s, len=%d, n=%v\n", k, len(v), n)
		if n > float32(0.5) {
			winner = v[0]
			break
		} else if n == float32(0.5) {
			//to-do: choose what leader has got to break the tie
			var leaderID string
			if localServer.GetLeaderPeer() == nil {
				leaderID = localServer.nodeID
			} else {
				leaderID = localServer.GetLeaderPeer().GetNodeID()
			}
			fmt.Println("Got a tie, and need to choose what the leader has for the winner of dirblock sig. leaderPeer=", leaderID)
			for _, d := range v {
				fmt.Println("leaderID=", leaderID, ", fed server id=", d.SourceNodeID)
				if leaderID == d.SourceNodeID {
					winner = d
					break
				}
			}
			// a tie without leader's dir block sig
			if winner != nil {
				break
			}
		}
	}
	if winner == nil {
		//risk: some nodes might get different number or set of dirblock signatures
		//or worse, some node could get a tie without leader's dirblock sig
		//request it from the leader
		// req := wire.NewDirBlockSigMsg()
		// localServer.GetLeaderPeer().pushGetDirBlockSig(req)
		// how to coordinate when the response comes ???
		//panic("No winner in dirblock signature comparison.")
		fmt.Println("No winner in dirblock signature comparison.")
	} else {
		fmt.Println("winner: ", spew.Sdump(winner))
	}

	// for followers, bypass the first block locally generated
	//if localServer.IsLeader() || newDBlock.Header.DBHeight != firstBlockHeight {
	go saveBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)
	//}
	return nil
}

// processAck validates the ack and adds it to processlist
// this is only for post-syncup followers need to deal with Ack
func processAck(msg *wire.MsgAck) error {
	fmt.Printf("processAck: %s\n", spew.Sdump(msg))
	// Validate the signiture
	bytes, err := msg.GetBinaryForSignature()
	if err != nil {
		fmt.Println("error in GetBinaryForSignature", err.Error())
		return err
	}
	//todo: must use the peer's, not server's, public key to verify signature here
	if !serverPubKey.Verify(bytes, &msg.Signature) {
		//to-do
		//return errors.New(fmt.Sprintf("Invalid signature in Ack = %s\n", spew.Sdump(msg)))
		fmt.Println("verify ack signature: FAILED")
	} else {
		fmt.Println("verify ack signature: SUCCESS")
	}
	_, latestHeight, _ := db.FetchBlockHeightCache()
	fmt.Printf("** ack.Height=%d, dchain.NextDBHeight=%d, db.FetchBlockHeightCache()=%d\n",
		msg.Height, dchain.NextDBHeight, latestHeight)

	missingMsg := fMemPool.addAck(msg)
	if missingMsg != nil {
		//todo: request missing acks from Leader
		//how to coordinate new processAck when missing acks come ???
		//
		fmt.Println("** missing msg: ", spew.Sdump(missingMsg))
	}

	var missingAcks []*wire.MsgAck
	// only check missing acks every minute
	if msg.IsEomAck() {
		missingAcks = fMemPool.getMissingMsgAck(msg)
		if len(missingAcks) > 0 {
			fmt.Printf("missing Acks total: %d\n", len(missingAcks)) //, spew.Sdump(missingAcks))
			//todo: request missing acks from Leader
			//how to coordinate new processAck when missing acks come ???
			//
		}
	}
	// go happy path for now. todo
	if msg.Type == wire.END_MINUTE_10 { //}&& missingMsg == nil && len(missingAcks) == 0 {
		fmt.Println("assembleFollowerProcessList")
		fMemPool.assembleFollowerProcessList(msg)
		//procLog.Infof("current ProcessList: %s", spew.Sdump(plMgr.MyProcessList))
	}
	// for firstBlockHeight, ususally there's some msg or ack missing
	// let's bypass the first one to give the follower time to round up.
	// todo: for this block (firstBlockHeight), we need to request for it. ???
	if msg.Type == wire.END_MINUTE_10 { //&& msg.Height != firstBlockHeight {
		// followers build Blocks
		fmt.Println("follower buildBlocks, height=", msg.Height)
		err = buildBlocks() //broadcast new dir block sig
		if err != nil {
			return err
		}
	}
	return nil
}

// processRevealEntry validates the MsgRevealEntry and adds it to processlist
func processRevealEntry(msg *wire.MsgRevealEntry) error {
	e := msg.Entry
	bin, _ := e.MarshalBinary()
	h, _ := wire.NewShaHash(e.Hash().Bytes())

	// Check if the chain id is valid
	if e.ChainID.IsSameAs(zeroHash) || e.ChainID.IsSameAs(dchain.ChainID) || e.ChainID.IsSameAs(achain.ChainID) ||
		e.ChainID.IsSameAs(ecchain.ChainID) || e.ChainID.IsSameAs(fchain.ChainID) {
		return fmt.Errorf("This entry chain is not supported: %s", e.ChainID.String())
	}

	if c, ok := commitEntryMap[e.Hash().String()]; ok {
		if chainIDMap[e.ChainID.String()] == nil {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("This chain is not supported: %s",
				msg.Entry.ChainID.String())
		}

		// Calculate the entry credits required for the entry
		cred, err := util.EntryCost(bin)
		if err != nil {
			return err
		}

		if c.Credits < cred {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("Credit needs to paid first before an entry is revealed: %s", e.Hash().String())
		}

		// Add the msg to the Mem pool
		fMemPool.addMsg(msg, h)

		// Add to MyPL if Server Node
		//if nodeMode == common.SERVER_NODE {
		if localServer.IsLeader() || localServer.isSingleServerMode() {
			if plMgr.IsMyPListExceedingLimit() {
				fmt.Println("Exceeding MyProcessList size limit!")
				return fMemPool.addOrphanMsg(msg, h)
			}

			ack, err := plMgr.AddToLeadersProcessList(msg, h, wire.ACK_REVEAL_ENTRY)
			if err != nil {
				return err
			}
			fmt.Printf("ACK_REVEAL_ENTRY: %s\n", spew.Sdump(ack))
			outMsgQueue <- ack
			//???
			delete(commitEntryMap, e.Hash().String())
		} else {
			//as follower
			h, _ := wire.NewShaHash(e.Hash().Bytes())
			fMemPool.addMsg(msg, h)
		}

		return nil
	} else if c, ok := commitChainMap[e.Hash().String()]; ok { //Reveal chain ---------------------------
		if chainIDMap[e.ChainID.String()] != nil {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("This chain is not supported: %s",
				msg.Entry.ChainID.String())
		}

		// add new chain to chainIDMap
		newChain := common.NewEChain()
		newChain.ChainID = e.ChainID
		newChain.FirstEntry = e
		chainIDMap[e.ChainID.String()] = newChain

		// Calculate the entry credits required for the entry
		cred, err := util.EntryCost(bin)
		if err != nil {
			return err
		}

		// 10 credit is additional for the chain creation
		if c.Credits < cred+10 {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("Credit needs to paid first before an entry is revealed: %s", e.Hash().String())
		}

		//validate chain id for the first entry
		expectedChainID := common.NewChainID(e)
		if !expectedChainID.IsSameAs(e.ChainID) {
			return fmt.Errorf("Invalid ChainID for entry: %s", e.Hash().String())
		}

		//validate chainid hash in the commitChain
		chainIDHash := common.DoubleSha(e.ChainID.Bytes())
		if !bytes.Equal(c.ChainIDHash.Bytes()[:], chainIDHash[:]) {
			return fmt.Errorf("RevealChain's chainid hash does not match with CommitChain: %s", e.Hash().String())
		}

		//validate Weld in the commitChain
		weld := common.DoubleSha(append(c.EntryHash.Bytes(), e.ChainID.Bytes()...))
		if !bytes.Equal(c.Weld.Bytes()[:], weld[:]) {
			return fmt.Errorf("RevealChain's weld does not match with CommitChain: %s", e.Hash().String())
		}
		return nil
	}
	return fmt.Errorf("No commit for entry")
}

// processCommitEntry validates the MsgCommitEntry and adds it to processlist
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

	if c.Credits > common.MAX_ENTRY_CREDITS {
		return fmt.Errorf("Commit entry exceeds the max entry credit limit:" + c.EntryHash.String())
	}

	// Check the entry credit balance
	if eCreditMap[string(c.ECPubKey[:])] < int32(c.Credits) {
		return fmt.Errorf("Not enough credits for CommitEntry")
	}

	// add to the commitEntryMap
	commitEntryMap[c.EntryHash.String()] = c

	// Server: add to MyPL
	//if nodeMode == common.SERVER_NODE {
	if localServer.IsLeader() || localServer.isSingleServerMode() {

		// deduct the entry credits from the eCreditMap
		eCreditMap[string(c.ECPubKey[:])] -= int32(c.Credits)

		h, _ := msg.Sha()
		if plMgr.IsMyPListExceedingLimit() {
			fmt.Println("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}

		ack, err := plMgr.AddToLeadersProcessList(msg, &h, wire.ACK_COMMIT_ENTRY)
		if err != nil {
			return err
		}
		fmt.Printf("ACK_COMMIT_ENTRY: %s\n", spew.Sdump(ack))
		outMsgQueue <- ack
	} else {
		//as follower
		h, _ := wire.NewShaHash(c.Hash().Bytes())
		fMemPool.addMsg(msg, h)
	}
	return nil
}

// processCommitChain validates the MsgCommitChain and adds it to processlist
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

	if c.Credits > common.MAX_CHAIN_CREDITS {
		return fmt.Errorf("Commit chain exceeds the max entry credit limit:" + c.EntryHash.String())
	}

	// Check the entry credit balance
	if eCreditMap[string(c.ECPubKey[:])] < int32(c.Credits) {
		return fmt.Errorf("Not enough credits for CommitChain")
	}

	// add to the commitChainMap
	commitChainMap[c.EntryHash.String()] = c

	// Server: add to MyPL
	//if nodeMode == common.SERVER_NODE {
	if localServer.IsLeader() || localServer.isSingleServerMode() {
		// deduct the entry credits from the eCreditMap
		eCreditMap[string(c.ECPubKey[:])] -= int32(c.Credits)

		h, _ := msg.Sha()

		if plMgr.IsMyPListExceedingLimit() {
			fmt.Println("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}

		ack, err := plMgr.AddToLeadersProcessList(msg, &h, wire.ACK_COMMIT_CHAIN)
		if err != nil {
			return err
		}
		fmt.Printf("ACK_COMMIT_CHAIN: %s\n", spew.Sdump(ack))
		outMsgQueue <- ack
	} else {
		//as follower
		h, _ := wire.NewShaHash(c.Hash().Bytes())
		fMemPool.addMsg(msg, h)
	}
	return nil
}

// processBuyEntryCredit validates the MsgCommitChain and adds it to processlist
func processBuyEntryCredit(msg *wire.MsgFactoidTX) error {
	// Update the credit balance in memory
	for _, v := range msg.Transaction.GetECOutputs() {
		pub := new([32]byte)
		copy(pub[:], v.GetAddress().Bytes())

		cred := int32(v.GetAmount() / uint64(FactoshisPerCredit))

		eCreditMap[string(pub[:])] += cred

	}

	h, _ := msg.Sha()
	if plMgr.IsMyPListExceedingLimit() {
		fmt.Println("Exceeding MyProcessList size limit!")
		return fMemPool.addOrphanMsg(msg, &h)
	}

	if _, err := plMgr.AddToLeadersProcessList(msg, &h, wire.ACK_FACTOID_TX); err != nil {
		return err
	}

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
				fmt.Println("Error in processing orphan msgCommitChain:" + err.Error())
				continue
			}
			delete(fMemPool.orphans, k)

		case wire.CmdCommitEntry:
			msgCommitEntry, _ := msg.(*wire.MsgCommitEntry)
			err := processCommitEntry(msgCommitEntry)
			if err != nil {
				fmt.Println("Error in processing orphan msgCommitEntry:" + err.Error())
				continue
			}
			delete(fMemPool.orphans, k)

		case wire.CmdRevealEntry:
			msgRevealEntry, _ := msg.(*wire.MsgRevealEntry)
			err := processRevealEntry(msgRevealEntry)
			if err != nil {
				fmt.Println("Error in processing orphan msgRevealEntry:" + err.Error())
				continue
			}
			delete(fMemPool.orphans, k)
		}
	}
	return nil
}

func buildRevealEntry(msg *wire.MsgRevealEntry) {
	chain := chainIDMap[msg.Entry.ChainID.String()]

	// store the new entry in db
	db.InsertEntry(msg.Entry)

	err := chain.NextBlock.AddEBEntry(msg.Entry)

	if err != nil {
		panic("Error while adding Entity to Block:" + err.Error())
	}

}

func buildIncreaseBalance(msg *wire.MsgFactoidTX) {
	t := msg.Transaction
	for i, ecout := range t.GetECOutputs() {
		ib := common.NewIncreaseBalance()

		pub := new([32]byte)
		copy(pub[:], ecout.GetAddress().Bytes())
		ib.ECPubKey = pub

		th := common.NewHash()
		th.SetBytes(t.GetHash().Bytes())
		ib.TXID = th

		cred := int32(ecout.GetAmount() / uint64(FactoshisPerCredit))
		ib.NumEC = uint64(cred)

		ib.Index = uint64(i)

		ecchain.NextBlock.AddEntry(ib)
	}
}

func buildCommitEntry(msg *wire.MsgCommitEntry) {
	ecchain.NextBlock.AddEntry(msg.CommitEntry)
}

func buildCommitChain(msg *wire.MsgCommitChain) {
	ecchain.NextBlock.AddEntry(msg.CommitChain)
}

func buildRevealChain(msg *wire.MsgRevealEntry) {
	chain := chainIDMap[msg.Entry.ChainID.String()]

	// Store the new chain in db
	db.InsertChain(chain)

	// Chain initialization
	initEChainFromDB(chain)

	// store the new entry in db
	db.InsertEntry(chain.FirstEntry)

	err := chain.NextBlock.AddEBEntry(chain.FirstEntry)

	if err != nil {
		panic(fmt.Sprintf(`Error while adding the First Entry to Block: %s`,
			err.Error()))
	}
}

// Loop through the Process List items and get the touched chains
// Put End-Of-Minute marker in the entry chains
func buildEndOfMinute(pl *consensus.ProcessList, pli *consensus.ProcessListItem) {
	tmpChains := make(map[string]*common.EChain)
	for _, v := range pl.GetPLItems()[:pli.Ack.Index] {
		if v.Ack.Type == wire.ACK_REVEAL_ENTRY ||
			v.Ack.Type == wire.ACK_REVEAL_CHAIN {
			cid := v.Msg.(*wire.MsgRevealEntry).Entry.ChainID.String()
			tmpChains[cid] = chainIDMap[cid]
		} else if wire.END_MINUTE_1 <= v.Ack.Type &&
			v.Ack.Type <= wire.END_MINUTE_10 {
			tmpChains = make(map[string]*common.EChain)
		}
	}
	for _, v := range tmpChains {
		v.NextBlock.AddEndOfMinuteMarker(pli.Ack.Type)
	}

	// Add it to the entry credit chain
	cbEntry := common.NewMinuteNumber()
	cbEntry.Number = pli.Ack.Type
	ecchain.NextBlock.AddEntry(cbEntry)

	// Add it to the admin chain
	abEntries := achain.NextBlock.ABEntries
	if len(abEntries) > 0 && abEntries[len(abEntries)-1].Type() != common.TYPE_MINUTE_NUM {
		achain.NextBlock.AddEndOfMinuteMarker(pli.Ack.Type)
	}
}

// build Genesis blocks
func buildGenesisBlocks() error {
	//Set the timestamp for the genesis block
	t, err := time.Parse(time.RFC3339, common.GENESIS_BLK_TIMESTAMP)
	if err != nil {
		panic("Not able to parse the genesis block time stamp")
	}
	dchain.NextBlock.Header.Timestamp = uint32(t.Unix() / 60)

	// Allocate the first two dbentries for ECBlock and Factoid block
	dchain.AddDBEntry(&common.DBEntry{}) // AdminBlock
	dchain.AddDBEntry(&common.DBEntry{}) // ECBlock
	dchain.AddDBEntry(&common.DBEntry{}) // Factoid block

	// Entry Credit Chain
	cBlock := newEntryCreditBlock(ecchain)
	procLog.Debugf("buildGenesisBlocks: cBlock=%s\n", spew.Sdump(cBlock.Header))
	dchain.AddECBlockToDBEntry(cBlock)
	exportECChain(ecchain)

	// Admin chain
	aBlock := newAdminBlock(achain)
	procLog.Debugf("buildGenesisBlocks: aBlock=%s\n", spew.Sdump(aBlock.Header))
	dchain.AddABlockToDBEntry(aBlock)
	exportAChain(achain)

	// factoid Genesis Address
	//fchain.NextBlock = block.GetGenesisFBlock(0, FactoshisPerCredit, 10, 200000000000)
	fchain.NextBlock = block.GetGenesisFBlock()
	FBlock := newFactoidBlock(fchain)
	procLog.Debugf("buildGenesisBlocks: fBlock.height=%ds\n", FBlock.GetDBHeight())
	dchain.AddFBlockToDBEntry(FBlock)
	exportFctChain(fchain)

	// Directory Block chain
	procLog.Debug("onto newDirectoryBlock in buildGenesisBlocks")
	dbBlock := newDirectoryBlock(dchain)
	exportDChain(dchain)

	// Check block hash if genesis block
	if dbBlock.DBHash.String() != common.GENESIS_DIR_BLOCK_HASH {
		//Panic for Milestone 1
		panic("\nGenesis block hash expected: " + common.GENESIS_DIR_BLOCK_HASH +
			"\nGenesis block hash found:    " + dbBlock.DBHash.String() + "\n")
	}

	initProcessListMgr()

	// saveBlocks will be done at the next EOM_1
	//go saveBlocks(dbBlock, aBlock, cBlock, FBlock, nil)
	return nil
}

// build blocks from all process lists
func buildBlocks() error {
	// build the Genesis blocks if the current height is 0
	fmt.Printf("buildBlocks(): dchain.NextDBHeight=%d, achain.NextDBHeight=%d, fchain.NextDBHeight=%d, ecchain.NextDBHeight=%d\n",
		dchain.NextDBHeight, achain.NextBlockHeight, fchain.NextBlockHeight, ecchain.NextBlockHeight)

	// Allocate the first three dbentries for Admin block, ECBlock and Factoid block
	dchain.AddDBEntry(&common.DBEntry{}) // AdminBlock
	dchain.AddDBEntry(&common.DBEntry{}) // ECBlock
	dchain.AddDBEntry(&common.DBEntry{}) // factoid

	if plMgr != nil {
		buildFromProcessList(plMgr.MyProcessList)
	}

	// Entry Credit Chain
	ecBlock := newEntryCreditBlock(ecchain)
	dchain.AddECBlockToDBEntry(ecBlock)

	// Admin chain
	aBlock := newAdminBlock(achain)
	dchain.AddABlockToDBEntry(aBlock)

	// Factoid chain
	fBlock := newFactoidBlock(fchain)
	dchain.AddFBlockToDBEntry(fBlock)

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
		newEBlocks = append(newEBlocks, eblock)
	}

	// Directory Block chain
	procLog.Debug("in buildBlocks")
	newDirectoryBlock(dchain) //sign dir block and broadcast it

	// should keep this process list for a while ????
	// re-initialize the process lit manager
	initProcessListMgr()

	// for leader / follower regime change
	fmt.Printf("buildBlocks(): It's a leader=%t or leaderElected=%t, SingleServerMode=%t, dbheight=%d, dchain.NextDBHeight=%d\n",
		localServer.IsLeader(), localServer.isLeaderElected, localServer.isSingleServerMode(), newDBlock.Header.DBHeight, dchain.NextDBHeight)

	if localServer.IsLeader() && !localServer.isSingleServerMode() {
		if dchain.NextDBHeight-1 == localServer.myLeaderPolicy.StartDBHeight+localServer.myLeaderPolicy.Term-1 {
			fmt.Println("buildBlocks(): Leader turn OFF BlockTimer. newDBlock.dbheight=", newDBlock.Header.DBHeight, ", dchain.NextDBHeight=", dchain.NextDBHeight)
		} else {
			fmt.Println("buildBlocks(): Leader RESTARTs BlockTimer. newDBlock.dbheight=", newDBlock.Header.DBHeight, ", dchain.NextDBHeight=", dchain.NextDBHeight)
		}
	}
	if localServer.isLeaderElected && !localServer.isSingleServerMode() {
		if dchain.NextDBHeight-1 == localServer.myLeaderPolicy.StartDBHeight-1 {
			fmt.Println("buildBlocks(): Leader-Elected turn ON BlockTimer. newDBlock.dbheight=", newDBlock.Header.DBHeight, ", dchain.NextDBHeight=", dchain.NextDBHeight)
		} else {
			fmt.Println("buildBlocks(): Leader-Elected KEEPs OFF BlockTimer. newDBlock.dbheight=", newDBlock.Header.DBHeight, ", dchain.NextDBHeight=", dchain.NextDBHeight)
		}
	}

	// should have a long-lasting block timer ???
	// Initialize timer for the new dblock, only for leader before regime change.
	if localServer.isSingleServerMode() ||
		localServer.IsLeader() && dchain.NextDBHeight-1 != localServer.myLeaderPolicy.StartDBHeight+localServer.myLeaderPolicy.Term-1 ||
		localServer.isLeaderElected && dchain.NextDBHeight-1 == localServer.myLeaderPolicy.StartDBHeight-1 {
		fmt.Println("@@@@ start BlockTimer ")
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextDBHeight,
			inMsgQueue:       inMsgQueue,
		}
		go timer.StartBlockTimer()
	}

	if localServer.IsLeader() || localServer.isLeaderElected {
		fmt.Println("buildBlocks(): send it to channel: localServer.latestDBHeight <- ", newDBlock.Header.DBHeight)
		localServer.latestDBHeight <- newDBlock.Header.DBHeight
	}
	return nil
}

// build blocks from a process lists
func buildFromProcessList(pl *consensus.ProcessList) error {
	for _, pli := range pl.GetPLItems() {
		//fmt.Println("buildFromProcessList: pli=", spew.Sdump(pli))
		if pli == nil {
			continue
		}
		if pli.Ack.Type == wire.ACK_COMMIT_CHAIN {
			buildCommitChain(pli.Msg.(*wire.MsgCommitChain))
		} else if pli.Ack.Type == wire.ACK_FACTOID_TX {
			buildIncreaseBalance(pli.Msg.(*wire.MsgFactoidTX))
		} else if pli.Ack.Type == wire.ACK_COMMIT_ENTRY {
			buildCommitEntry(pli.Msg.(*wire.MsgCommitEntry))
		} else if pli.Ack.Type == wire.ACK_REVEAL_CHAIN {
			buildRevealChain(pli.Msg.(*wire.MsgRevealEntry))
		} else if pli.Ack.Type == wire.ACK_REVEAL_ENTRY {
			buildRevealEntry(pli.Msg.(*wire.MsgRevealEntry))
		} else if wire.END_MINUTE_1 <= pli.Ack.Type && pli.Ack.Type <= wire.END_MINUTE_10 {
			buildEndOfMinute(pl, pli)
		}
	}

	return nil
}

// Seals the current open block, store it in db and create the next open block
func newEntryBlock(chain *common.EChain) *common.EBlock {
	// acquire the last block
	block := chain.NextBlock
	if block == nil {
		return nil
	}
	if len(block.Body.EBEntries) < 1 {
		procLog.Debug("No new entry found. No block created for chain: " + chain.ChainID.String())
		return nil
	}

	// Create the block and add a new block for new coming entries
	block.Header.EBHeight = dchain.NextDBHeight
	block.Header.EntryCount = uint32(len(block.Body.EBEntries))
	fmt.Println("newEntryBlock: block.Header.EBHeight = ", block.Header.EBHeight)

	chain.NextBlockHeight++
	var err error
	chain.NextBlock, err = common.MakeEBlock(chain, block)
	if err != nil {
		procLog.Debug("EntryBlock Error: " + err.Error())
		return nil
	}
	return block
}

// Seals the current open block, store it in db and create the next open block
func newEntryCreditBlock(chain *common.ECChain) *common.ECBlock {

	// acquire the last block
	block := chain.NextBlock
	if block.Header.EBHeight != chain.NextBlockHeight {
		// this is the first block after block sync up
		block.Header.EBHeight = chain.NextBlockHeight
	}

	if chain.NextBlockHeight != dchain.NextDBHeight {
		fmt.Println("Entry Credit Block height does not match Directory Block height: ", dchain.NextDBHeight)
	}

	block.BuildHeader()
	fmt.Println("newEntryCreditBlock: block.Header.EBHeight = ", block.Header.EBHeight)

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	var err error
	chain.NextBlock, err = common.NextECBlock(block)
	if err != nil {
		procLog.Debug("EntryCreditBlock Error: " + err.Error())
		return nil
	}
	chain.NextBlock.AddEntry(serverIndex)
	chain.BlockMutex.Unlock()
	newECBlock = block
	return block
}

// Seals the current open block, store it in db and create the next open block
func newAdminBlock(chain *common.AdminChain) *common.AdminBlock {

	// acquire the last block
	block := chain.NextBlock
	if block.Header.DBHeight != chain.NextBlockHeight {
		// this is the first block after block sync up
		block.Header.DBHeight = chain.NextBlockHeight
	}

	if chain.NextBlockHeight != dchain.NextDBHeight {
		fmt.Printf("Admin Block height does not match Directory Block height: ablock height=%d, dblock height=%d",
			chain.NextBlockHeight, dchain.NextDBHeight)
		//panic("Admin Block height does not match Directory Block height:" + string(dchain.NextDBHeight))
	}

	block.Header.MessageCount = uint32(len(block.ABEntries))
	block.Header.BodySize = uint32(block.MarshalledSize() - block.Header.MarshalledSize())
	_, err := block.PartialHash()
	if err != nil {
		panic(err)
	}
	_, err = block.LedgerKeyMR()
	if err != nil {
		panic(err)
	}
	fmt.Println("newAdminBlock: block.Header.EBHeight = ", block.Header.DBHeight)

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	chain.NextBlock, err = common.CreateAdminBlock(chain, block, 10)
	if err != nil {
		panic(err)
	}
	chain.BlockMutex.Unlock()
	newABlock = block
	return block
}

// Seals the current open block, store it in db and create the next open block
func newFactoidBlock(chain *common.FctChain) block.IFBlock {

	older := FactoshisPerCredit

	cfg := util.ReReadConfig()
	FactoshisPerCredit = cfg.App.ExchangeRate

	rate := fmt.Sprintf("Current Exchange rate is %v",
		strings.TrimSpace(fct.ConvertDecimal(FactoshisPerCredit)))
	if older != FactoshisPerCredit {

		orate := fmt.Sprintf("The Exchange rate was    %v\n",
			strings.TrimSpace(fct.ConvertDecimal(older)))

		cp.CP.AddUpdate(
			"Fee",    // tag
			"status", // Category
			"Entry Credit Exchange Rate Changed", // Title
			orate+rate,
			0)
	} else {
		cp.CP.AddUpdate(
			"Fee",                        // tag
			"status",                     // Category
			"Entry Credit Exchange Rate", // Title
			rate,
			0)
	}

	// acquire the last block
	currentBlock := chain.NextBlock
	fmt.Println("newFactoidBlock: block.Header.EBHeight = ", currentBlock.GetDBHeight())
	if currentBlock.GetDBHeight() != chain.NextBlockHeight {
		// this is the first block after block sync up
		currentBlock.SetDBHeight(chain.NextBlockHeight)
	}

	if chain.NextBlockHeight != dchain.NextDBHeight {
		fmt.Println("Factoid Block height does not match Directory Block height:" + strconv.Itoa(int(dchain.NextDBHeight)))
		//panic("Factoid Block height does not match Directory Block height:" + strconv.Itoa(int(dchain.NextDBHeight)))
	}

	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	common.FactoidState.SetFactoshisPerEC(FactoshisPerCredit)
	common.FactoidState.ProcessEndOfBlock2(chain.NextBlockHeight)
	chain.NextBlock = common.FactoidState.GetCurrentBlock()
	chain.BlockMutex.Unlock()
	newFBlock = currentBlock
	return currentBlock
}

// Seals the current open block, store it in db and create the next open block
func newDirectoryBlock(chain *common.DChain) *common.DirectoryBlock {
	fmt.Printf("enter newDirectoryBlock: chain.NextBlock.Height=%d, chain.NextDBHeight=%d\n",
		chain.NextBlock.Header.DBHeight, chain.NextDBHeight)
	// acquire the last block
	block := chain.NextBlock
	if block.Header.DBHeight != chain.NextDBHeight {
		// this is the first block after block sync up
		block.Header.DBHeight = chain.NextDBHeight
	}

	if devNet {
		block.Header.NetworkID = common.NETWORK_ID_TEST
	} else {
		block.Header.NetworkID = common.NETWORK_ID_EB
	}

	// Create the block add a new block for new coming entries
	chain.BlockMutex.Lock()
	block.Header.BlockCount = uint32(len(block.DBEntries))
	// Calculate Merkle Root for FBlock and store it in header
	if block.Header.BodyMR == nil {
		block.Header.BodyMR, _ = block.BuildBodyMR()
		//  Factoid1 block not in the right place...
	}
	block.IsSealed = true
	chain.AddDBlockToDChain(block)
	chain.NextDBHeight++
	chain.NextBlock, _ = common.CreateDBlock(chain, block, 10)
	chain.BlockMutex.Unlock()
	fmt.Printf("newDirectoryBlock: new dbBlock=%s\n", spew.Sdump(block.Header))
	fmt.Println("after creating new dir block, dchain.NextDBHeight=", chain.NextDBHeight)

	newDBlock = block
	block.DBHash, _ = common.CreateHash(block)
	block.BuildKeyMerkleRoot()

	// send out dir block sig first
	if dchain.NextDBHeight > 1 {
		SignDirectoryBlock(block)
	}
	return block
}

// save all blocks and anchor dir block if it's the leader
func saveBlocks(dblock *common.DirectoryBlock, ablock *common.AdminBlock,
	ecblock *common.ECBlock, fblock block.IFBlock, eblocks []*common.EBlock) error {
	// save blocks to database in a signle transaction ???
	fmt.Println("saveBlocks: height=", dblock.Header.DBHeight)
	db.ProcessFBlockBatch(fblock)
	exportFctBlock(fblock)
	fmt.Println("Save Factoid Block: block " + strconv.FormatUint(uint64(fblock.GetDBHeight()), 10))

	db.ProcessABlockBatch(ablock)
	exportABlock(ablock)
	fmt.Println("Save Admin Block: block " + strconv.FormatUint(uint64(ablock.Header.DBHeight), 10))

	db.ProcessECBlockBatch(ecblock)
	exportECBlock(ecblock)
	fmt.Println("Save EntryCreditBlock: block " + strconv.FormatUint(uint64(ecblock.Header.EBHeight), 10))

	db.ProcessDBlockBatch(dblock)
	db.InsertDirBlockInfo(common.NewDirBlockInfoFromDBlock(dblock))
	fmt.Println("Save DirectoryBlock: block " + strconv.FormatUint(uint64(dblock.Header.DBHeight), 10))

	for _, eblock := range eblocks {
		db.ProcessEBlockBatch(eblock)
		exportEBlock(eblock)
		fmt.Println("Save EntryBlock: block " + strconv.FormatUint(uint64(eblock.Header.EBSequence), 10))
	}

	binary, _ := dblock.MarshalBinary()
	commonHash := common.Sha(binary)
	db.UpdateBlockHeightCache(dblock.Header.DBHeight, commonHash)
	db.UpdateNextBlockHeightCache(dchain.NextDBHeight)
	exportDBlock(dblock)
	fmt.Println("saveBlocks: done=", dblock.Header.DBHeight)

	placeAnchor(dblock)
	fMemPool.resetDirBlockSigPool()
	return nil
}

// SignDirectoryBlock signs the directory block and broadcast it
func SignDirectoryBlock(newdb *common.DirectoryBlock) error {
	// Only Servers can write the anchor to Bitcoin network
	if nodeMode == common.SERVER_NODE && dchain.NextDBHeight > 0 { //&& localServer.isLeader {
		fmt.Println("SignDirectoryBlock: dchain.NextDBHeight: ", dchain.NextDBHeight) // 11th minute
		// get the previous directory block from db
		// since saveBlocks happens at 11th minute, almost 1 minute after buildBlocks
		// so the latest block height in database should be dchain.NextDBHeight - 2
		// and newdb.DBHeight should be dchain.NextDBHeight - 1
		dbBlock, _ := db.FetchDBlockByHeight(dchain.NextDBHeight - 1)
		//????
		if dbBlock == nil {
			dbBlock, _ = db.FetchDBlockByHeight(dchain.NextDBHeight - 2)
		}
		if dbBlock == nil {
			dbBlock, _ = db.FetchDBlockByHeight(dchain.NextDBHeight - 3)
		}
		fmt.Printf("SignDirBlock: dbBlock from db=%s\n", spew.Sdump(dbBlock.Header))
		fmt.Printf("SignDirBlock: new dbBlock=%s\n", spew.Sdump(newdb.Header))
		dbHeaderBytes, _ := dbBlock.Header.MarshalBinary()
		identityChainID := common.NewHash() // 0 ID for milestone 1 ????
		sig := serverPrivKey.Sign(dbHeaderBytes)
		achain.NextBlock.AddABEntry(common.NewDBSignatureEntry(identityChainID, sig))

		//create and broadcast dir block sig message
		dbHeaderBytes, _ = newdb.Header.MarshalBinary()
		h := common.Hash{}
		//h.SetBytes(dbHeaderBytes[:]) //wrong
		hash := common.Sha(dbHeaderBytes)
		h.SetBytes(hash.GetBytes())
		sig = serverPrivKey.Sign(dbHeaderBytes)
		msg := &wire.MsgDirBlockSig{
			DBHeight:     newdb.Header.DBHeight,
			DirBlockHash: h, //????
			Sig:          sig,
			SourceNodeID: localServer.nodeID, // use peer.nodeID ???
		}
		outMsgQueue <- msg
		fmt.Println("my own: addDirBlockSig: ", spew.Sdump(msg))
		fMemPool.addDirBlockSig(msg)
	}
	return nil
}

// Place an anchor into btc
func placeAnchor(dblock *common.DirectoryBlock) error {
	if localServer.isLeader || localServer.isSingleServerMode() {
		fmt.Printf("placeAnchor: height=%d, leader=%s\n", dblock.Header.DBHeight, localServer.nodeID)
		anchor.UpdateDirBlockInfoMap(common.NewDirBlockInfoFromDBlock(dblock))
		go anchor.SendRawTransactionToBTC(dblock.KeyMR, dblock.Header.DBHeight)
	}
	return nil
}

//IsDChainInSync shows if dchain is in sync with network dbheight.
func IsDChainInSync() bool {
	_, latestHeight, _ := db.FetchBlockHeightCache()
	if dchain.NextDBHeight == uint32(latestHeight)+1 {
		return true
	}
	return false
}
