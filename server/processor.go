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
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	"github.com/FactomProject/ed25519"
	//cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wire"
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

	dchain  *common.DChain     //Directory Block Chain
	ecchain *common.ECChain    //Entry Credit Chain
	achain  *common.AdminChain //Admin Chain
	fchain  *common.FctChain   // factoid Chain

	newDBlock  *common.DirectoryBlock
	newABlock  *common.AdminBlock
	newFBlock  block.IFBlock
	newECBlock *common.ECBlock
	newEBlocks []*common.EBlock

	commitChainMap = make(map[string]*common.CommitChain, 0)
	commitChainMapBackup map[string]*common.CommitChain
	
	commitEntryMap = make(map[string]*common.CommitEntry, 0)
	commitEntryMapBackup map[string]*common.CommitEntry

	chainIDMap       map[string]*common.EChain // ChainIDMap with chainID string([32]byte) as key
	chainIDMapBackup map[string]*common.EChain //previous block bakcup - ChainIDMap with chainID string([32]byte) as key

	eCreditMap       map[string]int32 // eCreditMap with public key string([32]byte) as key, credit balance as value
	eCreditMapBackup map[string]int32 // backup from previous block - eCreditMap with public key string([32]byte) as key, credit balance as value

	fMemPool *ftmMemPool
	plMgr    *consensus.ProcessListMgr

	//Server Private key and Public key for milestone 1
	serverPrivKey common.PrivateKey
	serverPubKey  common.PublicKey

	// FactoshisPerCredit is .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents
	FactoshisPerCredit                 uint64
	blockSyncing                       bool
	firstBlockHeight                   uint32 // the DBHeight of the first block being built by follower after sync up
	doneSetFollowersCointbaseTimeStamp bool
	doneSentCandidateMsg               bool
	leaderCrashed 										 bool

	zeroHash                = common.NewHash()
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
	//cp.CP.SetPort(factomConfig.Controlpanel.Port)
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
			//fmt.Println("Error found in validating directory blocks: " + err.Error())
		} else {
			dchain.IsValidated = false
		}
	}
}

// StartProcessor is started from factomd
func StartProcessor(wg *sync.WaitGroup, quit chan struct{}) {
	//fmt.Println("StartProcessor: blockSyncing=", blockSyncing)

	wg.Add(1)
	defer func() {
		wg.Done()
	}()

	if localServer.IsLeader() {
		if dchain.NextDBHeight == 0 {
			buildGenesisBlocks()
			procLog.Debug("after creating genesis block: dchain.NextDBHeight=", dchain.NextDBHeight)
		}
		// sign the last dir block for next admin block
		signDirBlockForAdminBlock(dchain.Blocks[dchain.NextDBHeight-1])

		// Initialize timer for the open dblock before processing messages
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextDBHeight,
			inMsgQueue:       inMsgQueue,
		}
		go timer.StartBlockTimer()
	} //else {
	// process the blocks and entries downloaded from peers
	// this is needed for clients and followers and leader when sync up
	//fmt.Println("StartProcessor: validateAndStoreBlocks")
	go validateAndStoreBlocks(fMemPool, db, dchain)
	//}

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
				//fmt.Println("StartProcessor: case of MsgInt_DirBlock: ", spew.Sdump(msg))
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
				//fmt.Println("StartProcessor: case of Message (outMsgQueue): ", spew.Sdump(msg))
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
				//panic(fmt.Sprintf("bad outMsgQueue message received: %v", msg))
				fmt.Printf("bad outMsgQueue message received: %#v\n", msg)
			}
		case <-quit:
			return
		}
	}
}

// Serve incoming msg from inMsgQueue
func serveMsgRequest(msg wire.FtmInternalMsg) error {
	//procLog.Infof("serveMsgRequest: %s", spew.Sdump(msg))
	switch msg.Command() {
	case wire.CmdCommitChain:
		msgCommitChain, ok := msg.(*wire.MsgCommitChain)
		if ok && msgCommitChain.IsValid() {
			h := msgCommitChain.CommitChain.GetSigHash().Bytes()
			t := msgCommitChain.CommitChain.GetMilliTime() / 1000
			if !IsTSValid(h, t) {
				return fmt.Errorf("Timestamp invalid on Commit Chain")
			}

			// broadcast it to other federate servers only if it's new to me
			hash, _ := msgCommitChain.Sha()
			fmt.Printf("CmdCommitChain: msgCommitChain=%s, hash=%s\n", spew.Sdump(msgCommitChain), hash.String())
			if fMemPool.haveMsg(hash) {
				fmt.Printf("CmdCommitChain: already in mempool. msgCommitChain=%s, hash=%s\n", 
					spew.Sdump(msgCommitChain), hash.String())
				return nil
			}
			fMemPool.addMsg(msgCommitChain, &hash)
			outMsgQueue <- msgCommitChain

			err := processCommitChain(msgCommitChain)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + spew.Sdump(msg))
		}

	case wire.CmdCommitEntry:
		msgCommitEntry, ok := msg.(*wire.MsgCommitEntry)
		if ok && msgCommitEntry.IsValid() {
			h := msgCommitEntry.CommitEntry.GetSigHash().Bytes()
			t := msgCommitEntry.CommitEntry.GetMilliTime() / 1000
			if !IsTSValid(h, t) {
				return fmt.Errorf("Timestamp invalid on Commit Entry")
			}

			// broadcast it to other federate servers only if it's new to me
			hash, _ := msgCommitEntry.Sha()
			fmt.Printf("CmdCommitEntry: msgCommitEntry=%s, hash=%s\n", spew.Sdump(msgCommitEntry), hash.String())
			if fMemPool.haveMsg(hash) {
				fmt.Printf("CmdCommitEntry: already in mempool. msgCommitEntry=%s, hash=%s\n", 
					spew.Sdump(msgCommitEntry), hash.String())
				return nil
			}
			fMemPool.addMsg(msgCommitEntry, &hash)
			outMsgQueue <- msgCommitEntry

			err := processCommitEntry(msgCommitEntry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + spew.Sdump(msg))
		}

	case wire.CmdRevealEntry:
		msgRevealEntry, ok := msg.(*wire.MsgRevealEntry)
		if ok && msgRevealEntry.IsValid() {
			// broadcast it to other federate servers only if it's new to me
			h, _ := msgRevealEntry.Sha()
			fmt.Printf("CmdRevealEntry: msgRevealEntry=%s, hash=%s\n", spew.Sdump(msgRevealEntry), h.String())
			if fMemPool.haveMsg(h) {
				fmt.Printf("CmdRevealEntry: already in mempool. msgRevealEntry=%s, hash=%s\n", 
					spew.Sdump(msgRevealEntry), h.String())
				return nil
			}
			fMemPool.addMsg(msgRevealEntry, &h)
			outMsgQueue <- msgRevealEntry

			err := processRevealEntry(msgRevealEntry)
			if err != nil {
				return err
			}
		} else {
			return errors.New("Error in processing msg:" + spew.Sdump(msg))
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
		// to simplify this, use the next wire.EndMinute1 to trigger signature comparison. ???
		fMemPool.addDirBlockSig(dbs)

	case wire.CmdInt_EOM:
		// this is only for leader. as followers have no blocktimer
		// followers EOM will be driven by Ack of this EOM.
		fmt.Printf("in case.CmdInt_EOM: localServer.IsLeader=%v\n", localServer.IsLeader())
		msgEom, _ := msg.(*wire.MsgInt_EOM)
		if leaderCrashed {
			miss := checkMissingLeaderEOMs(msgEom)
			if len(miss) > 0 {
				for _, m := range miss {
					fmt.Println("process missing eom: ", m.EOM_Type)
					processLeaderEOM(m)
				}
			}
			leaderCrashed = false
		}
		return processLeaderEOM(msgEom)

	case wire.CmdFactoidTX:
		msgFactoidTX, ok := msg.(*wire.MsgFactoidTX)
		if !ok || !msgFactoidTX.IsValid() {
			return errors.New("Error in processing msg:" + fmt.Sprintf("%+v", msg))
		}
		return processFactoidTX(msgFactoidTX)

	default:
		return errors.New("Message type unsupported:" + fmt.Sprintf("%+v", msg))
	}

	return nil
}

func processLeaderEOM(msgEom *wire.MsgInt_EOM) error {
	if nodeMode != common.SERVER_NODE {
		return nil
	}
	var singleServerMode = localServer.isSingleServerMode()
	fmt.Println("processLeaderEOM: federate servers #: ", localServer.FederateServerCount(),
		", Non-candidate federate servers #: ", localServer.NonCandidateServerCount(),
		", singleServerMode=", singleServerMode)

	// to simplify this, for leader & followers, use the next wire.EndMinute1
	// to trigger signature comparison of last round.
	// todo: when to start? can NOT do this for the first EOM_1 ???
	if msgEom.EOM_Type == wire.EndMinute2 {
		if !localServer.isSingleServerMode() && fMemPool.lenDirBlockSig() > 0 {
			go processDirBlockSig()
		} else {
			// newDBlock is the genesis block or normal single server mode
			if newDBlock != nil {
				go saveBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)
			}
		}
	}
	common.FactoidState.EndOfPeriod(int(msgEom.EOM_Type))

	ack, err := plMgr.AddToLeadersProcessList(msgEom, nil, msgEom.EOM_Type, 
		dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
	if err != nil {
		return err
	}
	//???
	if ack.ChainID == nil {
		ack.ChainID = dchain.ChainID
	}
	fmt.Printf("processLeaderEOM: leader sending ack=%s\n", spew.Sdump(ack))
	outMsgQueue <- ack

	if msgEom.EOM_Type == wire.EndMinute10 {
		fmt.Println("leader's ProcessList: ", spew.Sdump(plMgr.MyProcessList))
		err = buildBlocks() //broadcast new dir block sig
		if err != nil {
			return err
		}
	}
	/*
		cp.CP.AddUpdate(
			"MinMark",  // tag
			"status",   // Category
			"Progress", // Title
			fmt.Sprintf("End of Minute %v\n", msgEom.EOM_Type)+ // Message
				fmt.Sprintf("Directory Block Height %v", dchain.NextDBHeight),
			0)*/
	return nil
}

// processDirBlockSig check received MsgDirBlockMsg in mempool and
// decide which dir block to save to database and for anchor.
func processDirBlockSig() error {
	if nodeMode != common.SERVER_NODE {
		return nil
	}
	_, latestHeight, _ := db.FetchBlockHeightCache()
	if uint32(latestHeight) == dchain.NextDBHeight-1 {
		fmt.Printf("processDirBlockSig: block already saved: db.latestDBHeight=%d, dchain.NextDBHeight=%d\n", 
			latestHeight, dchain.NextDBHeight)
		return nil
	} 
	if fMemPool.lenDirBlockSig() == 0 {
		fmt.Println("no dir block sig in mempool.")
		return nil
	}

	leadPeer := localServer.GetLeaderPeer()		//.leaderPeer
	if localServer.latestLeaderSwitchDBHeight == dchain.NextDBHeight {
		leadPeer = localServer.GetPrevLeaderPeer()		//.prevLeaderPeer
	}
	leaderID := localServer.nodeID
	if leadPeer != nil {
		leaderID = leadPeer.nodeID
	}
	
	dgsMap, leaderDirBlockSig, myDirBlockSig := fMemPool.getDirBlockSigMap(leaderID)

	fmt.Println("leaderID=", leaderID)
	if myDirBlockSig != nil {
		fmt.Println("myNodeID=", myDirBlockSig.SourceNodeID)
	}

	// Todo: This is not very good, because leader is not always the winner
	//
	// this is a candidate who just bypassed building block and
	// needs to download newly generated blocks up to firstBlockHeight
	if firstBlockHeight == dchain.NextDBHeight-1 {
		fmt.Printf("processDirBlockSig: I am a candidate just turned follower, "+
			"leadPeer=%s, dbheight=%d\n", leadPeer, dchain.NextDBHeight-1)
		if leaderDirBlockSig != nil {
			fmt.Println("download it from leaderPeer with known hash. ", leadPeer)
			downloadNewDirBlock(leadPeer, leaderDirBlockSig.DirBlockHash, dchain.NextDBHeight-1)
		} else {
			fmt.Println("download it from leaderPeer with unknown hash. ", leadPeer)
			downloadNewDirBlock(leadPeer, *zeroHash, dchain.NextDBHeight-1)
		}
		return nil
	} 

	totalServerNum := localServer.NonCandidateServerCount()
	fmt.Printf("processDirBlockSig: By EOM_1, there're %d dirblock signatures "+
		"arrived out of %d non-candidate federate servers.\n", fMemPool.lenDirBlockSig(), totalServerNum)

	var needDownload, needFromNonLeader bool
	var winner *wire.MsgDirBlockSig
	for k, v := range dgsMap {
		n := float32(len(v)) / float32(totalServerNum)
		fmt.Printf("key=%s, len=%d, n=%v\n", k, len(v), n)
		if n >= float32(1.0) {
			fmt.Println("A full consensus !")
			winner = myDirBlockSig
			needDownload = false
			break
		} else if n > float32(0.5) {
			var hasMe, hasLeader bool
			for _, d := range v {
				if leaderID == d.SourceNodeID {
					hasLeader = true
				} 
				if myDirBlockSig != nil && myDirBlockSig.SourceNodeID == d.SourceNodeID {
					hasMe = true
				}
			}
			if hasMe {
				needDownload = false
				winner = myDirBlockSig
			} else {
				needDownload = true
				if hasLeader {
					winner = leaderDirBlockSig
				} else {
					needFromNonLeader = true
					winner = v[0]
				}
			}
			fmt.Printf("A majority !  hasLeader=%v, hasMe=%v, needDownload=%v, "+
				"needFromNonLeader=%v, winner=%s\n", hasLeader, hasMe, needDownload, needFromNonLeader, spew.Sdump(winner))
			break
		} else if n == float32(0.5) {
			if leaderDirBlockSig != nil {
				if myDirBlockSig != nil && leaderDirBlockSig.Equals(myDirBlockSig) {
					winner = myDirBlockSig
					needDownload = false
					fmt.Printf("A tie with leader & me. needDownload=%v, winner=%s\n", needDownload, spew.Sdump(winner))
				} else {
					winner = leaderDirBlockSig
					needDownload = true
					fmt.Printf("A tie with leader but not me. needDownload=%v, winner=%s\n", needDownload, spew.Sdump(winner))
				}
			} else {
				needDownload = true
				needFromNonLeader = true
				// winner is leaderDirBlockSig but is nil here
				fmt.Printf("A tie without leader. needDownload=%v, winner= leaderDirBlockSig(ni)\n", needDownload)
			}
			break
		}
	}

	if needDownload {
		fmt.Println("needDownload is true")
		if !needFromNonLeader {
			fmt.Println("downloadNewDirBlock: from leader")
			downloadNewDirBlock(leadPeer, leaderDirBlockSig.DirBlockHash, dchain.NextDBHeight-1)
		} else {
			if winner != nil {
				var peer *peer
				for _, fs := range localServer.federateServers {
					if fs.Peer != nil && fs.Peer.nodeID == winner.SourceNodeID {
						peer = fs.Peer
						break
					}
				}
			  fmt.Println("downloadNewDirBlock: from non-leader")
				downloadNewDirBlock(peer, winner.DirBlockHash, dchain.NextDBHeight-1)
			} else {
			  fmt.Println("downloadNewDirBlock: from leader, for unknow hash")
				downloadNewDirBlock(leadPeer, *zeroHash, dchain.NextDBHeight-1)
			}
		}
	} else {
		go saveBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)
	}
	return nil
}

func downloadNewDirBlock(p *peer, hash common.Hash, height uint32) {
	fmt.Printf("downloadNewDirBlock: height=%d, hash=%s, peer=%s\n", height, hash.String(), p)
	//if zeroHash.IsSameAs(&hash) {
	if bytes.Compare(zeroHash.Bytes(), hash.Bytes()) == 0 {
		iv := wire.NewInvVectHeight(wire.InvTypeFactomGetDirData, hash, int64(height))
		gdmsg := wire.NewMsgGetFactomData()
		gdmsg.AddInvVectHeight(iv)
		// what happens when p as leaderPeer just crashed?
		p.QueueMessage(gdmsg, nil)
	} else {
		sha, _ := wire.NewShaHash(hash.Bytes())
		iv := wire.NewInvVect(wire.InvTypeFactomDirBlock, sha)
		gdmsg := wire.NewMsgGetDirData()
		gdmsg.AddInvVect(iv)
		// what happens when p as leaderPeer just crashed?
		p.QueueMessage(gdmsg, nil)
	}
}

// processAckPeerMsg validates the ack and adds it to processlist
// this is only for post-syncup followers need to deal with Ack
func processAckPeerMsg(ack *ackMsg) ([]*wire.MsgMissing, error) {
	//fmt.Printf("processAckPeerMsg: %s\n", spew.Sdump(ack))
	// Validate the signiture
	bytes, err := ack.ack.GetBinaryForSignature()
	if err != nil {
		fmt.Println("error in GetBinaryForSignature", err.Error())
		return nil, err
	}
	if !ack.peer.pubKey.Verify(bytes, &ack.ack.Signature) {
		//to-do
		//return errors.New(fmt.Sprintf("Invalid signature in Ack = %s\n", spew.Sdump(ack)))
		fmt.Println("verify ack signature: FAILED")
	}
	return processAckMsg(ack.ack)
}

// processAckMsg validates the ack and adds it to processlist
// this is only for post-syncup followers need to deal with Ack
func processAckMsg(ack *wire.MsgAck) ([]*wire.MsgMissing, error) {
	if nodeMode != common.SERVER_NODE || localServer == nil || localServer.IsLeader() {
		return nil, nil
	}
	_, latestHeight, _ := db.FetchBlockHeightCache()
	fmt.Printf("processAckMsg: Ack.Height=%d, Ack.Index=%d, dchain.NextDBHeight=%d, db.latestDBHeight=%d, blockSyncing=%v\n",
		ack.Height, ack.Index, dchain.NextDBHeight, latestHeight, blockSyncing)
	
	//dchain.NextDBHeight is the dir block height for the network
	//update it with ack height from the leader if necessary
	if dchain.NextDBHeight < ack.Height {
		dchain.NextDBHeight = ack.Height
	}
	//switch from block syncup to block build
	if dchain.NextDBHeight == uint32(latestHeight)+1 && blockSyncing {
		blockSyncing = false
		firstBlockHeight = ack.Height
		fmt.Println("processAckMsg: reset blockSyncing=false, bypass building this block: firstBlockHeight=", firstBlockHeight)
	}
	if !doneSetFollowersCointbaseTimeStamp && ack.CoinbaseTimestamp > 0 {
		fchain.NextBlock.SetCoinbaseTimestamp(ack.CoinbaseTimestamp)
		doneSetFollowersCointbaseTimeStamp = true
		fmt.Printf("processAckMsg: reset follower's CoinbaseTimestamp: %d\n", ack.CoinbaseTimestamp)
	}
	myPeer := localServer.GetMyFederateServer().Peer
	if blockSyncing || firstBlockHeight == dchain.NextDBHeight {
		myPeer.nodeState = wire.NodeCandidate
	} else if myPeer.IsCandidate() {		// in case myPeer is already leaderElect or even leader
		myPeer.nodeState = wire.NodeFollower
	}
	
	// reset factoid state after the firstBlockHeight
	// this is when a candidate is officially turned into a follower
	// and it starts to generate blocks.
	if firstBlockHeight == dchain.NextDBHeight-1 && !doneSentCandidateMsg {
		// when sync up, FactoidState.CurrentBlock is using the last block being synced-up
		// as it's needed for balance update.
		// when done sync up, reset its current block, for EndOfPeriod
		common.FactoidState.SetCurrentBlock(fchain.NextBlock)
		sig := localServer.privKey.Sign([]byte(string(dchain.NextDBHeight) + localServer.nodeID))
		m := wire.NewMsgCandidate(firstBlockHeight, localServer.nodeID, sig)
		outMsgQueue <- m
		doneSentCandidateMsg = true
		fmt.Println("processAckMsg: sending MsgCandidate: ", spew.Sdump(m))
		// update this federate server
		localServer.GetMyFederateServer().FirstAsFollower = firstBlockHeight
	}

	// to simplify this, for leader & followers, use the next wire.EndMinute1
	// to trigger signature comparison of last round.
	// todo: when to start? can NOT do this for the first EOM_1 ???
	if ack.Type == wire.EndMinute2 {
		// need to bypass the first block of newly-joined follower, 
		// either this is 11th minute: ack.NextDBlockHeight-1 == firstBlockHeight
		// or is 1st minute: 
		fmt.Println("processAckMsg: before go processDirBlockSig: firstBlockHeight=", firstBlockHeight, 
			", DirBlockSig.len=", fMemPool.lenDirBlockSig(), ", isCandidate=", localServer.IsCandidate(), 
			", ack=", spew.Sdump(ack))
		if fMemPool.lenDirBlockSig() > 0 && ack.Height > firstBlockHeight && !localServer.IsCandidate() { // the block is not saved yet
			go processDirBlockSig()
		}
	}

	// for candidate server, bypass block building
	if localServer.IsCandidate() {
		fmt.Println("processAckMsg: is candidate, return, no block building.")
		return nil, nil
	}

	if ack.IsEomAck() {
		fmt.Println("processAckMsg: follower's EndOfPeriod: type=", int(ack.Type))
		common.FactoidState.EndOfPeriod(int(ack.Type))
	}

	var missingAcks []*wire.MsgMissing
	missingMsg := fMemPool.addAck(ack)
	// only check missing acks every minute
	if ack.IsEomAck() {
		missingAcks = fMemPool.getMissingMsgAck(ack)
		if len(missingAcks) > 0 {
			fmt.Printf("processAckMsg: missing Acks total: %d\n", len(missingAcks)) //, spew.Sdump(missingAcks))
		}
		if dchain.NextBlock.Header.Timestamp == 0 && ack.DBlockTimestamp > 0 {
			dchain.NextBlock.Header.Timestamp = ack.DBlockTimestamp
			fmt.Printf("processAckMsg: reset follower's DBlockTimestamp: %d\n", ack.DBlockTimestamp)
		}
	}
	if missingMsg != nil {
		fmt.Println("processAckMsg: missing MSG: ", spew.Sdump(missingMsg))
		missingAcks = append(missingAcks, missingMsg)
	}
	if len(missingAcks) > 0 {
		fmt.Printf("processAckMsg: missing Acks %s\n", spew.Sdump(missingAcks))
		sort.Sort(wire.ByMsgIndex(missingAcks))
		if ack.Type != wire.EndMinute10 {
			return missingAcks, nil
		}
	}
	if ack.Type == wire.EndMinute10 {
		fmt.Println("processAckMsg: assembleFollowerProcessList")
		err := fMemPool.assembleFollowerProcessList(ack)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("processAckMsg: follower ProcessList: %s\n", spew.Sdump(plMgr.MyProcessList))
		// followers build Blocks
		fmt.Println("processAckMsg: follower buildBlocks, height=", ack.Height)
		err = buildBlocks() //broadcast new dir block sig
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// processRevealEntry validates the MsgRevealEntry and adds it to processlist
func processRevealEntry(msg *wire.MsgRevealEntry) error {
	if nodeMode != common.SERVER_NODE {
		return nil
	}
	e := msg.Entry
	bin, _ := e.MarshalBinary()
	hash, _ := msg.Sha()
	h := &hash
	//h, _ := wire.NewShaHash(e.Hash().Bytes())

	// Check if the chain id is valid
	if e.ChainID.IsSameAs(zeroHash) || e.ChainID.IsSameAs(dchain.ChainID) || e.ChainID.IsSameAs(achain.ChainID) ||
		e.ChainID.IsSameAs(ecchain.ChainID) || e.ChainID.IsSameAs(fchain.ChainID) {
		return fmt.Errorf("This entry chain is not supported: %s", e.ChainID.String())
	}

	if c, ok := commitEntryMap[e.Hash().String()]; ok {
		fmt.Println("processRevealEntry: commitEntryMap=", spew.Sdump(commitEntryMap))
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

		if localServer.IsLeader() || localServer.isSingleServerMode() {
			if plMgr.IsMyPListExceedingLimit() {
				fmt.Println("Exceeding MyProcessList size limit!")
				return fMemPool.addOrphanMsg(msg, h)
			}
			ack, err := plMgr.AddToLeadersProcessList(msg, h, wire.AckRevealEntry, dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
			if err != nil {
				return err
			}
			fmt.Printf("AckRevealEntry: %s\n", spew.Sdump(ack))
			outMsgQueue <- ack
			//???
			delete(commitEntryMap, e.Hash().String())
		}
		return nil

	} else if c, ok := commitChainMap[e.Hash().String()]; ok { //Reveal chain ---------------------------
		fmt.Println("processRevealEntry: commitChainMap=", spew.Sdump(commitChainMap))
		if chainIDMap[e.ChainID.String()] != nil {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("This chain already exists: %s",
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

		if localServer.IsLeader() || localServer.isSingleServerMode() {
			if plMgr.IsMyPListExceedingLimit() {
				fmt.Println("Exceeding MyProcessList size limit!")
				return fMemPool.addOrphanMsg(msg, h)
			}
			ack, err := plMgr.AddToLeadersProcessList(msg, h, wire.AckRevealChain, dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
			if err != nil {
				return err
			}
			fmt.Printf("AckRevealChain: %s\n", spew.Sdump(ack))
			outMsgQueue <- ack
			//???
			delete(commitChainMap, e.Hash().String())
		}
		return nil
	}
	return fmt.Errorf("No commit for entry")
}

// processCommitEntry validates the MsgCommitEntry and adds it to processlist
func processCommitEntry(msg *wire.MsgCommitEntry) error {
	if nodeMode != common.SERVER_NODE {
		return nil
	}
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

	// deduct the entry credits from the eCreditMap
	eCreditMap[string(c.ECPubKey[:])] -= int32(c.Credits)

	h, _ := msg.Sha()
	if localServer.IsLeader() || localServer.isSingleServerMode() {
		if plMgr.IsMyPListExceedingLimit() {
			fmt.Println("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}
		ack, err := plMgr.AddToLeadersProcessList(msg, &h, wire.AckCommitEntry, dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
		if err != nil {
			return err
		}
		fmt.Printf("AckCommitEntry: %s\n", spew.Sdump(ack))
		outMsgQueue <- ack
	}
	return nil
}

// processCommitChain validates the MsgCommitChain and adds it to processlist
func processCommitChain(msg *wire.MsgCommitChain) error {
	if nodeMode != common.SERVER_NODE {
		return nil
	}
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

	// deduct the entry credits from the eCreditMap
	eCreditMap[string(c.ECPubKey[:])] -= int32(c.Credits)

	h, _ := msg.Sha()
	if localServer.IsLeader() || localServer.isSingleServerMode() {
		if plMgr.IsMyPListExceedingLimit() {
			fmt.Println("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}
		ack, err := plMgr.AddToLeadersProcessList(msg, &h, wire.AckCommitChain, dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
		if err != nil {
			return err
		}
		fmt.Printf("AckCommitChain: %s\n", spew.Sdump(ack))
		outMsgQueue <- ack
	}
	return nil
}

// processFactoidTX validates the MsgFactoidTX and adds it to processlist
func processFactoidTX(msg *wire.MsgFactoidTX) error {
	if nodeMode != common.SERVER_NODE {
		return nil
	}
	// prevent replay attacks
	hash := msg.Transaction.GetSigHash().Bytes()
	t := int64(msg.Transaction.GetMilliTimestamp() / 1000)

	if !IsTSValid(hash, t) {
		return fmt.Errorf("Timestamp invalid on Factoid Transaction")
	}

	// broadcast it to other federate servers only if it's new to me
	h, _ := msg.Sha()
	if fMemPool.haveMsg(h) {
		fmt.Println("processFactoidTX: already in mempool. ", spew.Sdump(msg))
		return nil
	}
	outMsgQueue <- msg
	fMemPool.addMsg(msg, &h)

	tx := msg.Transaction
	txnum := 0
	if common.FactoidState.GetCurrentBlock() == nil {
		return fmt.Errorf("FactoidState.GetCurrentBlock() is nil")
	}
	txnum = len(common.FactoidState.GetCurrentBlock().GetTransactions())
	err := common.FactoidState.AddTransaction(txnum, tx)
	if err != nil {
		return err
	}
	// Update the credit balance in memory
	for _, v := range msg.Transaction.GetECOutputs() {
		pub := new([32]byte)
		copy(pub[:], v.GetAddress().Bytes())
		cred := int32(v.GetAmount() / uint64(FactoshisPerCredit))
		eCreditMap[string(pub[:])] += cred
	}

	if localServer.IsLeader() || localServer.isSingleServerMode() {
		if plMgr.IsMyPListExceedingLimit() {
			fmt.Println("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}
		ack, err := plMgr.AddToLeadersProcessList(msg, &h, wire.AckFactoidTx, dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
		if err != nil {
			return err
		}
		if ack == nil {
			return fmt.Errorf("processFactoidTX: ack is nil")
		}
		fmt.Printf("AckFactoidTx: %s\n", spew.Sdump(ack))
		outMsgQueue <- ack
	}
	return nil
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

func buildRevealEntry(msg *wire.MsgRevealEntry) error {
	chain := chainIDMap[msg.Entry.ChainID.String()]
	if chain == nil {
		return fmt.Errorf("buildRevealEntry: chain is nil for chainID=" + msg.Entry.ChainID.String())
	}
	if chain.NextBlock == nil {
		chain.NextBlock = common.NewEBlock()
	}

	// store the new entry in db
	db.InsertEntry(msg.Entry)

	err := chain.NextBlock.AddEBEntry(msg.Entry)

	if err != nil {
		//panic("Error while adding Entity to Block:" + err.Error())
		fmt.Println("buildRevealEntry: Error while adding Entity to Block:" + err.Error())
		return err
	}
	return nil
}

func buildRevealChain(msg *wire.MsgRevealEntry) error {
	chain := chainIDMap[msg.Entry.ChainID.String()]
	if chain == nil {
		return fmt.Errorf("buildRevealChain: chain is nil for chainID=" + msg.Entry.ChainID.String())
	}
	if chain.NextBlock == nil {
		chain.NextBlock = common.NewEBlock()
	}

	// Store the new chain in db
	db.InsertChain(chain)

	// Chain initialization. let initEChains and storeBlocksFromMemPool take care of it
	// initEChainFromDB(chain)

	// store the new entry in db
	db.InsertEntry(chain.FirstEntry)

	err := chain.NextBlock.AddEBEntry(chain.FirstEntry)

	if err != nil {
		fmt.Printf(`buildRevealChain: Error while adding the First Entry to EBlock: %s`, err.Error())
		return err
	}
	return nil
}

// Loop through the Process List items and get the touched chains
// Put End-Of-Minute marker in the entry chains
func buildEndOfMinute(pl *consensus.ProcessList, pli *consensus.ProcessListItem) {
	tmpChains := make(map[string]*common.EChain)
	for _, v := range pl.GetPLItems()[:pli.Ack.Index] {
		if v == nil {
			fmt.Println("process list item v is nil. ")
			continue
		} else if v.Ack == nil {
			fmt.Println("process list item ack v.ack is nil. ")
			continue
		}
		if v.Ack.Type == wire.AckRevealEntry ||
			v.Ack.Type == wire.AckRevealChain {
			cid := v.Msg.(*wire.MsgRevealEntry).Entry.ChainID.String()
			tmpChains[cid] = chainIDMap[cid]
		} else if wire.EndMinute1 <= v.Ack.Type &&
			v.Ack.Type <= wire.EndMinute10 {
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
		fmt.Println("Not able to parse the genesis block time stamp")
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
	procLog.Debugf("buildGenesisBlocks: fBlock.height=%d\n", FBlock.GetDBHeight())
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
	fmt.Printf("buildBlocks: dchain.NextDBHeight=%d, achain.NextDBHeight=%d, fchain.NextDBHeight=%d, ecchain.NextDBHeight=%d\n",
		dchain.NextDBHeight, achain.NextBlockHeight, fchain.NextBlockHeight, ecchain.NextBlockHeight)

	// Allocate the first three dbentries for Admin block, ECBlock and Factoid block
	dchain.AddDBEntry(&common.DBEntry{}) // AdminBlock
	dchain.AddDBEntry(&common.DBEntry{}) // ECBlock
	dchain.AddDBEntry(&common.DBEntry{}) // factoid

	if plMgr != nil {
		err := buildFromProcessList(plMgr.MyProcessList)
		fmt.Println("buildBlocks: ", err.Error())
	}

	var errStr string

	// Entry Credit Chain
	ecBlock := newEntryCreditBlock(ecchain)
	if ecBlock == nil {
		errStr = "buildBlocks: ecBlock is nil; "
		fmt.Println(errStr)
	} else {
		dchain.AddECBlockToDBEntry(ecBlock)
		bytes, _ := ecBlock.MarshalBinary()
		fmt.Printf("buildBlocks: ecBlock=%s\n", spew.Sdump(ecBlock))
		fmt.Printf("buildBlocks: ecBlock=%s\n", hex.EncodeToString(bytes))
	}

	// Admin chain
	aBlock := newAdminBlock(achain)
	if aBlock == nil {
		errStr = errStr + "ablock is nil; "
	} else {
		dchain.AddABlockToDBEntry(aBlock)
		bytes, _ := aBlock.MarshalBinary()
		fmt.Printf("buildBlocks: adminBlock=%s\n", spew.Sdump(aBlock))
		fmt.Printf("buildBlocks: adminBlock=%s\n", hex.EncodeToString(bytes))
	}

	// Factoid chain
	fBlock := newFactoidBlock(fchain)
	if fBlock == nil {
		errStr = errStr + "fBlock is nil; "
	} else {
		dchain.AddFBlockToDBEntry(fBlock)
		bytes, _ := fBlock.MarshalBinary()
		fmt.Printf("buildBlocks: factoidBlock=%s\n", fBlock)
		fmt.Printf("buildBlocks: factoidBlock=%s\n", hex.EncodeToString(bytes))
	}

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
			newEBlocks = append(newEBlocks, eblock)
		}
	}

	// Directory Block chain
	if aBlock != nil && ecBlock != nil && fBlock != nil {
		dBlock := newDirectoryBlock(dchain) //sign dir block and broadcast it
		if dBlock == nil {
			errStr = errStr + "fBlock is nil; "
		} else {
			// todo: relay new blocks to all candidates if i'm the leader
			// and to my clients
			bytes, _ := dBlock.MarshalBinary()
			//fmt.Printf("buildBlocks: dirBlock=%s\n", spew.Sdump(dBlock))
			fmt.Printf("buildBlocks: dirBlock=%s\n", hex.EncodeToString(bytes))
		}
	}

	// should keep this process list for a while ????
	backupKeyMapData()
	// remove msg of current process list from mempool
	fMemPool.cleanUpMemPool()	
	initProcessListMgr()
	
	// for leader / follower regime change
	fmt.Printf("buildBlocks: It's a leader=%t or leaderElected=%t, SingleServerMode=%t, dchain.NextDBHeight=%d",
		localServer.IsLeader(), localServer.IsLeaderElect(), localServer.isSingleServerMode(), dchain.NextDBHeight)
	if newDBlock != nil && newDBlock.Header != nil {
		fmt.Printf(", dbheight=%d\n", newDBlock.Header.DBHeight)
	}

	if localServer.IsLeader() && !localServer.isSingleServerMode() {
		if dchain.NextDBHeight-1 == localServer.myLeaderPolicy.StartDBHeight+localServer.myLeaderPolicy.Term-1 {
			fmt.Println("buildBlocks: Leader turn OFF BlockTimer. dchain.NextDBHeight=", dchain.NextDBHeight)
			if newDBlock != nil && newDBlock.Header != nil {
				fmt.Println("newDBlock.dbheight=", newDBlock.Header.DBHeight)
			}
		} else {
			fmt.Println("buildBlocks: Leader RESTARTs BlockTimer. dchain.NextDBHeight=", dchain.NextDBHeight)
			if newDBlock != nil && newDBlock.Header != nil {
				fmt.Println("newDBlock.dbheight=", newDBlock.Header.DBHeight)
			}
		}
	}
	if localServer.IsLeaderElect() && !localServer.isSingleServerMode() {
		if dchain.NextDBHeight-1 == localServer.myLeaderPolicy.StartDBHeight-1 {
			fmt.Println("buildBlocks: Leader-Elected turn ON BlockTimer. dchain.NextDBHeight=", dchain.NextDBHeight)
			if newDBlock != nil && newDBlock.Header != nil {
				fmt.Println("newDBlock.dbheight=", newDBlock.Header.DBHeight)
			}
		} else {
			fmt.Println("buildBlocks: Leader-Elected KEEPs OFF BlockTimer. dchain.NextDBHeight=", dchain.NextDBHeight)
			if newDBlock != nil && newDBlock.Header != nil {
				fmt.Println("newDBlock.dbheight=", newDBlock.Header.DBHeight)
			}
		}
	}

	// should have a long-lasting block timer ???
	// Initialize timer for the new dblock, only for leader before regime change.
	if localServer.isSingleServerMode() ||
		localServer.IsLeader() && dchain.NextDBHeight-1 != localServer.myLeaderPolicy.StartDBHeight+localServer.myLeaderPolicy.Term-1 ||
		localServer.IsLeaderElect() && dchain.NextDBHeight-1 == localServer.myLeaderPolicy.StartDBHeight-1 {
		fmt.Println("@@@@ start BlockTimer ")
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextDBHeight,
			inMsgQueue:       inMsgQueue,
		}
		go timer.StartBlockTimer()
	}

	if localServer.IsLeader() || localServer.IsLeaderElect() {
		fmt.Println("buildBlocks: send it to channel: localServer.latestDBHeight <- ", newDBlock.Header.DBHeight)
		localServer.latestDBHeight <- newDBlock.Header.DBHeight
	}

	// relay stale messages left in mempool and orphan pool.
	// fMemPool.relayStaleMessages()
	return nil
}

// build blocks from a process lists
func buildFromProcessList(pl *consensus.ProcessList) error {
	for _, pli := range pl.GetPLItems() {
		//fmt.Println("buildFromProcessList: pli=", spew.Sdump(pli))
		if pli == nil {
			continue
		}
		if pli.Ack.Type == wire.AckCommitChain {
			buildCommitChain(pli.Msg.(*wire.MsgCommitChain))
		} else if pli.Ack.Type == wire.AckFactoidTx {
			buildIncreaseBalance(pli.Msg.(*wire.MsgFactoidTX))
		} else if pli.Ack.Type == wire.AckCommitEntry {
			buildCommitEntry(pli.Msg.(*wire.MsgCommitEntry))
		} else if pli.Ack.Type == wire.AckRevealChain {
			err := buildRevealChain(pli.Msg.(*wire.MsgRevealEntry))
			if err != nil {
				return err
			}
		} else if pli.Ack.Type == wire.AckRevealEntry {
			err := buildRevealEntry(pli.Msg.(*wire.MsgRevealEntry))
			if err != nil {
				return err
			}
		} else if wire.EndMinute1 <= pli.Ack.Type && pli.Ack.Type <= wire.EndMinute10 {
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
		procLog.Debug("newEntryBlock: No new entry found. No block created for chain: " + chain.ChainID.String())
		return nil
	}

	fmt.Printf("newEntryBlock: block.Header.EBHeight =%d, EBSequenc=%d, dchain.NextDBHeight=%d, block=%s\n ",
		block.Header.EBHeight, block.Header.EBSequence, dchain.NextDBHeight, spew.Sdump(block))

	if block.Header.EBSequence != chain.NextBlockHeight {
		// this is the first block after block sync up
		block.Header.EBSequence = chain.NextBlockHeight
		block.Header.ChainID = chain.ChainID
		prev, err := db.FetchEBlockByHeight(chain.ChainID, chain.NextBlockHeight-1)
		fmt.Println("newEntryBlock: prev=", spew.Sdump(prev))
		if err != nil {
			fmt.Println("newEntryBlock: error in db.FetchEBlockByHeight", err.Error())
			return nil
		}
		if prev == nil {
			fmt.Println("newEntryBlock from db: prev == nil")
			//check in mempool for this dir block
			prev = fMemPool.getEBlock(chain.NextBlockHeight - 1)
		}
		if prev == nil {
			fmt.Println("newEntryBlock from mempool: prev == nil")
			return nil
		}
		block.Header.PrevLedgerKeyMR, err = prev.Hash()
		if err != nil {
			fmt.Println("newEntryBlock: ", err.Error())
			return nil
		}
		block.Header.PrevKeyMR, err = prev.KeyMR()
		if err != nil {
			fmt.Println("newEntryBlock: ", err.Error())
			return nil
		}
	}

	// Create the block and add a new block for new coming entries
	block.Header.EBHeight = dchain.NextDBHeight
	block.Header.EntryCount = uint32(len(block.Body.EBEntries))
	fmt.Printf("newEntryBlock: block.Header.EBHeight =%d, EBSequenc=%d, dchain.NextDBHeight=%d\n ",
		block.Header.EBHeight, block.Header.EBSequence, dchain.NextDBHeight)

	chain.NextBlockHeight++
	var err error
	chain.NextBlock, err = common.MakeEBlock(chain, block)
	if err != nil {
		procLog.Debug("EntryBlock Error: " + err.Error())
		return nil
	}
	return block
}

// Note: on directory block heights 22881, 22884, 22941, 22950, 22977, and 22979,
// the ECBlock height fell behind DBHeight by 1, and
// then it fell behind by 2 on block 23269.
// In other words, there're two ecblocks of 22880, 22882, 22938, 22946,
// 22972, 22973, 23261 & 23262.
// ECBlocks:  22879, 22880, 22880, 22881, 22882, 22882, ..., 22938, 22938, ..., 22946, 22946, ..., 22972, 22972, 22973, 22973, ..., 23260, 23261, 23262, 23261, 23262, 23263, ...
// DirBlocks: 22879, 22880, 22881, 22882, 22883, 22884, ..., 22940, 22941, ..., 22949, 22950, ..., 22976, 22977, 22978, 22979, ..., 23266, 23267, 23268, 23269, 23270, 23271, ...
func newEntryCreditBlock(chain *common.ECChain) *common.ECBlock {
	// acquire the last block
	block := chain.NextBlock
	fmt.Printf("newEntryCreditBlock: block.Header.EBHeight =%d, chain.NextBlockHeight=%d\n ", block.Header.EBHeight, chain.NextBlockHeight)
	// do not intend to recreate the existing ec chain here
	// just simply make it work for echeight > 23263
	if block.Header.EBHeight > 23263 && block.Header.EBHeight >= chain.NextBlockHeight-8 {
		chain.NextBlockHeight = chain.NextBlockHeight - 8
		fmt.Printf("after adjust chain.NextBlockHeight - 8: block.Header.EBHeight =%d, chain.NextBlockHeight=%d\n ", block.Header.EBHeight, chain.NextBlockHeight)
	}
	if block.Header.EBHeight != chain.NextBlockHeight {
		// this is the first block after block sync up
		block.Header.EBHeight = chain.NextBlockHeight
		prev, err := db.FetchECBlockByHeight(chain.NextBlockHeight - 1)
		if err != nil {
			fmt.Println("newEntryCreditBlock: error in db.FetchECBlockByHeight", err.Error())
		}
		if prev == nil {
			fmt.Println("newEntryCreditBlock from db: prev == nil")
			//check in mempool for this dir block
			prev = fMemPool.getECBlock(chain.NextBlockHeight - 1)
		}
		if prev == nil {
			fmt.Println("newEntryCreditBlock from mempool: prev == nil")
			return nil
		}
		fmt.Println("newEntryCreditBlock: prev=", spew.Sdump(prev.Header))
		block.Header.PrevHeaderHash, err = prev.HeaderHash()
		if err != nil {
			fmt.Println("newEntryCreditBlock: ", err.Error())
		}
		block.Header.PrevLedgerKeyMR, err = prev.Hash()
		if err != nil {
			fmt.Println("newEntryCreditBlock: ", err.Error())
		}
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
	fmt.Printf("newAdminBlock: block.Header.DBHeight=%d, chain.NextBlockHeight=%d\n ", block.Header.DBHeight, chain.NextBlockHeight)
	if block.Header.DBHeight != chain.NextBlockHeight {
		// this is the first block after block sync up
		block.Header.DBHeight = chain.NextBlockHeight
		prev, err := db.FetchABlockByHeight(chain.NextBlockHeight - 1)
		if err != nil {
			fmt.Println("newAdminBlock: error in db.FetchABlockByHeight", err.Error())
		}
		if prev == nil {
			fmt.Println("newAdminBlock from db: prev == nil")
			//check in mempool for this dir block
			prev = fMemPool.getABlock(chain.NextBlockHeight - 1)
		}
		if prev == nil {
			fmt.Println("newAdminBlock from mempool: prev == nil")
			return nil
		}

		block.Header.PrevLedgerKeyMR, err = prev.LedgerKeyMR()
		if err != nil {
			fmt.Println("newAdminBlock: error in creating LedgerKeyMR", err.Error())
		}
		prevDB, err := db.FetchDBlockByHeight(dchain.NextDBHeight - 1)
		if err != nil {
			fmt.Println("newAdminBlock: error in db.FetchDBlockByHeight", err.Error())
		}
		fmt.Println("newAdminBlock: get prev DirBlock for aBlock signature: prev dirBlock = ", spew.Sdump(prevDB))
		signDirBlockForAdminBlock(prevDB)
		block.AddEndOfMinuteMarker(wire.EndMinute1)
	}

	if chain.NextBlockHeight != dchain.NextDBHeight {
		fmt.Printf("Admin Block height does not match Directory Block height: ablock height=%d, dblock height=%d\n",
			chain.NextBlockHeight, dchain.NextDBHeight)
		//panic("Admin Block height does not match Directory Block height:" + string(dchain.NextDBHeight))
	}

	block.Header.MessageCount = uint32(len(block.ABEntries))
	block.Header.BodySize = uint32(block.MarshalledSize() - block.Header.MarshalledSize())
	_, err := block.PartialHash()
	if err != nil {
		//panic(err)
		fmt.Println("newAdminBlock: error in creating block PartialHash.")
	}
	_, err = block.LedgerKeyMR()
	if err != nil {
		//panic(err)
		fmt.Println("newAdminBlock: error in creating block LedgerKeyMR.")
	}
	fmt.Println("newAdminBlock: block.Header.EBHeight = ", block.Header.DBHeight)

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	chain.NextBlock, err = common.CreateAdminBlock(chain, block, 10)
	if err != nil {
		//panic(err)
		fmt.Println("newAdminBlock: error in creating CreateAdminBlock.")
	}
	chain.BlockMutex.Unlock()
	newABlock = block
	return block
}

// Seals the current open block, store it in db and create the next open block
func newFactoidBlock(chain *common.FctChain) block.IFBlock {
	cfg := util.ReReadConfig()
	FactoshisPerCredit = cfg.App.ExchangeRate
	/*
		older := FactoshisPerCredit
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
	*/

	// acquire the last block
	currentBlock := chain.NextBlock
	fmt.Println("newFactoidBlock: block.Header.EBHeight = ", currentBlock.GetDBHeight(), ", chain.NextBlockHeight=", chain.NextBlockHeight)
	if currentBlock.GetDBHeight() != chain.NextBlockHeight {
		// this is the first block after block sync up
		currentBlock.SetDBHeight(chain.NextBlockHeight)
		prev, err := db.FetchFBlockByHeight(chain.NextBlockHeight - 1)
		//fmt.Println("newFactoidBlock: prev=", spew.Sdump(prev))
		if err != nil {
			fmt.Println("newFactoidBlock: error in db.FetchFBlockByHeight", err.Error())
		}
		if prev == nil {
			fmt.Println("newFactoidBlock from db: prev == nil")
			//check in mempool for this dir block
			prev = fMemPool.getFBlock(chain.NextBlockHeight - 1)
		}
		if prev == nil {
			fmt.Println("newFactoidBlock from mempool: prev == nil")
			return nil
		}
		currentBlock.SetExchRate(FactoshisPerCredit)
		currentBlock.SetPrevKeyMR(prev.GetKeyMR().Bytes())
		currentBlock.SetPrevLedgerKeyMR(prev.GetLedgerKeyMR().Bytes())
		currentBlock.SetCoinbaseTimestamp(plMgr.MyProcessList.GetCoinbaseTimestamp())
	}

	if chain.NextBlockHeight != dchain.NextDBHeight {
		fmt.Println("Factoid Block height does not match Directory Block height:" + strconv.Itoa(int(dchain.NextDBHeight)))
		//panic("Factoid Block height does not match Directory Block height:" + strconv.Itoa(int(dchain.NextDBHeight)))
	}
	fmt.Println("newFactoidBlock: block.Header.EBHeight = ", currentBlock.GetDBHeight())

	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	common.FactoidState.SetFactoshisPerEC(FactoshisPerCredit)
	common.FactoidState.ProcessEndOfBlock2(chain.NextBlockHeight)
	chain.NextBlock = common.FactoidState.GetCurrentBlock()
	chain.BlockMutex.Unlock()
	newFBlock = currentBlock
	doneSetFollowersCointbaseTimeStamp = false
	return currentBlock
}

// Seals the current open block, store it in db and create the next open block
func newDirectoryBlock(chain *common.DChain) *common.DirectoryBlock {
	// acquire the last block
	block := chain.NextBlock
	if block.Header.DBHeight != chain.NextDBHeight {
		// this is the first block after block sync up
		block.Header.DBHeight = chain.NextDBHeight
		prev, err := db.FetchDBlockByHeight(chain.NextDBHeight - 1)
		if err != nil {
			fmt.Println("newDirectoryBlock: error in db.FetchDBlockByHeight: ", err.Error())
		}
		if prev == nil {
			fmt.Println("newDirectoryBlock from db: prev == nil")
			//check in mempool for this dir block
			prev = fMemPool.getDirBlock(chain.NextDBHeight - 1)
		}
		if prev == nil {
			fmt.Println("newDirectoryBlock from mempool: prev == nil")
			return nil
		}
		//fmt.Println("newDirectoryBlock: prev=", spew.Sdump(prev))
		block.Header.PrevLedgerKeyMR, err = common.CreateHash(prev)
		if err != nil {
			fmt.Println("newDirectoryBlock: error in creating LedgerKeyMR", err.Error())
		}
		prev.BuildKeyMerkleRoot()
		block.Header.PrevKeyMR = prev.KeyMR
	}
	fmt.Printf("newDirectoryBlock: chain.NextBlock.Height=%d, chain.NextDBHeight=%d\n",
		chain.NextBlock.Header.DBHeight, chain.NextDBHeight)

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
	//fmt.Printf("newDirectoryBlock: new dbBlock=%s\n", spew.Sdump(block.Header))
	//fmt.Println("after creating new dir block, dchain.NextDBHeight=", chain.NextDBHeight)

	newDBlock = block
	block.DBHash, _ = common.CreateHash(block)
	block.BuildKeyMerkleRoot()

	// send out dir block sig first
	if dchain.NextDBHeight > 1 && block != nil {
		SignDirectoryBlock(block)
	}
	return block
}

// save all blocks and anchor dir block if it's the leader
func saveBlocks(dblock *common.DirectoryBlock, ablock *common.AdminBlock,
	ecblock *common.ECBlock, fblock block.IFBlock, eblocks []*common.EBlock) error {
	/*
	// in case of leader crashed and a new leader elected, excluding genesis block
	_, latestHeight, _ := db.FetchBlockHeightCache()
	if latestHeight > 0 && uint32(latestHeight) == dchain.NextDBHeight-1 {
		fmt.Printf("saveBlocks: block already saved: db.latestDBHeight=%d, dchain.NextDBHeight=%d\n", 
			latestHeight, dchain.NextDBHeight)
		return nil
	} */
	
	// in case of any block building failure, dblock will be nil. 
	// so download them immediately
	if dblock == nil || ablock == nil || ecblock == nil || fblock == nil {
		peer := localServer.GetLeaderPeer()		//.leaderPeer
		if peer == nil {
			peer = localServer.SyncPeer()
			if peer == nil {
				for _, fs := range localServer.federateServers {
					if !fs.Peer.IsCandidate() && fs.Peer.nodeID != localServer.nodeID {
						peer = fs.Peer
						break
					}
				}
			}
		}
		downloadNewDirBlock(peer, *zeroHash, dchain.NextDBHeight-1)
		return nil
	}

	db.ProcessDBlockBatch(dblock)
	db.ProcessFBlockBatch(fblock)
	db.ProcessECBlockBatch(ecblock)
	db.ProcessABlockBatch(ablock)
	db.InsertDirBlockInfo(common.NewDirBlockInfoFromDBlock(dblock))
	for _, eblock := range eblocks {
		if eblock == nil {
			continue
		}
		db.ProcessEBlockBatch(eblock)
	}

	binary, _ := dblock.MarshalBinary()
	commonHash := common.Sha(binary)
	db.UpdateBlockHeightCache(dblock.Header.DBHeight, commonHash)
	db.UpdateNextBlockHeightCache(dchain.NextDBHeight)
	fmt.Println("saveBlocks: done=", dblock.Header.DBHeight)
	
	placeAnchor(dblock)

	relayToCandidates()
	relayToClients()

	exportBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)	

	fMemPool.resetDirBlockSigPool(dblock.Header.DBHeight)
	newDBlock, newABlock, newECBlock, newFBlock, newEBlocks = nil, nil, nil, nil, nil
	return nil
}

/*
// save all blocks and anchor dir block if it's the leader
func saveBlocks(dblock *common.DirectoryBlock, ablock *common.AdminBlock,
	ecblock *common.ECBlock, fblock block.IFBlock, eblocks []*common.EBlock) error {
	// in case of any block building failure, download them immediately
	hash := zeroHash
	if dblock != nil {
		binaryDblock, err := dblock.MarshalBinary()
		if err != nil {
			return err
		}
		hash = common.Sha(binaryDblock)
	}
	h := common.Hash{}
	h.SetBytes(hash.GetBytes())
	if dblock == nil || ablock == nil || ecblock == nil || fblock == nil {
		peer := localServer.leaderPeer
		if peer == nil {
			for _, fs := range localServer.federateServers {
				if fs.Peer != nil {
					peer = fs.Peer
					break
				}
			}
		}
		downloadNewDirBlock(peer, h, dchain.NextDBHeight)
		return nil
	}

	db.StartBatch()

	// save blocks to database in a signle transaction ???
	fmt.Println("saveBlocks: height=", dblock.Header.DBHeight)
	err := db.ProcessFBlockMultiBatch(fblock)
	if err != nil {
		return err
	}

	err = db.ProcessABlockMultiBatch(ablock)
	if err != nil {
		return err
	}

	err = db.ProcessECBlockMultiBatch(ecblock)
	if err != nil {
		return err
	}

	err = db.ProcessDBlockMultiBatch(dblock)
	if err != nil {
		return err
	}
	err = db.InsertDirBlockInfoMultiBatch(common.NewDirBlockInfoFromDBlock(dblock))
	if err != nil {
		return err
	}

	for _, eblock := range eblocks {
		err = db.ProcessEBlockMultiBatch(eblock)
		if eblock == nil {
			continue
		}
		if err != nil {
			return err
		}
	}

	err = db.EndBatch()
	if err != nil {
		return err
	}

	binary, err := dblock.MarshalBinary()
	if err != nil {
		return err
	}
	commonHash := common.Sha(binary)
	db.UpdateBlockHeightCache(dblock.Header.DBHeight, commonHash)
	db.UpdateNextBlockHeightCache(dchain.NextDBHeight)
	fmt.Println("saveBlocks: done=", dblock.Header.DBHeight)

	placeAnchor(dblock)
	fMemPool.resetDirBlockSigPool(dblock.Header.DBHeight)

	// export blocks here for less database lock time
	exportBlocks(newDBlock, newABlock, newECBlock, newFBlock, newEBlocks)
	return nil
}*/

func exportBlocks(dblock *common.DirectoryBlock, ablock *common.AdminBlock,
	ecblock *common.ECBlock, fblock block.IFBlock, eblocks []*common.EBlock) {
	exportFctBlock(fblock)
	fmt.Println("export Factoid Block: block " + strconv.FormatUint(uint64(fblock.GetDBHeight()), 10))
	exportABlock(ablock)
	fmt.Println("export Admin Block: block " + strconv.FormatUint(uint64(ablock.Header.DBHeight), 10))
	exportECBlock(ecblock)
	fmt.Println("export EntryCreditBlock: block " + strconv.FormatUint(uint64(ecblock.Header.EBHeight), 10))
	exportDBlock(dblock)
	fmt.Println("export DirectoryBlock: block " + strconv.FormatUint(uint64(dblock.Header.DBHeight), 10))
	for _, eblock := range eblocks {
		exportEBlock(eblock)
		fmt.Println("export EntryBlock: block " + strconv.FormatUint(uint64(eblock.Header.EBSequence), 10))
	}
}

func relayToCandidates() {
	_, candidates := localServer.nonCandidateServers()
	fmt.Println("relayToCandidates: len(candidates)=", len(candidates))
	for _, candidate := range candidates {
		fmt.Println("relayToCandidates: ", candidate)
		relayNewBlocks(candidate.Peer) 
	}
}

func relayToClients() {
	fmt.Println("relayToClients: len=", len(localServer.clientPeers))
	for _, client := range localServer.clientPeers {
		fmt.Println("relayToClients: client=", client)
		relayNewBlocks(client)
	}
}

func relayNewBlocks(p *peer) {
	fmt.Println("relayNewBlocks: ", p)
	msgd := wire.NewMsgDirBlock()
	msgd.DBlk = newDBlock
	p.QueueMessage(msgd, nil) 
	/*
	msgf := wire.NewMsgFBlock()
	msgf.SC = newFBlock
	p.QueueMessage(msgf, nil) 
	
	msga := wire.NewMsgABlock()
	msga.ABlk = newABlock
	p.QueueMessage(msga, nil) 

	msgc := wire.NewMsgECBlock()
	msgc.ECBlock = newECBlock
	p.QueueMessage(msgc, nil) 

	for _, eblock := range newEBlocks {
		if eblock == nil {
			continue
		}
		msge := wire.NewMsgEBlock()
		msge.EBlk = eblock
		p.QueueMessage(msge, nil) 

		for _, ebEntry := range eblock.Body.EBEntries {
			//Skip the minute markers
			if ebEntry.IsMinuteMarker() {
				continue
			}
			entry, err := db.FetchEntryByHash(ebEntry)
			if entry != nil && err == nil {
				msgr := wire.NewMsgEntry()
				msgr.Entry = entry
				p.QueueMessage(msgr, nil) 
			}
		}
	}*/
}

// signDirBlockForAdminBlock signs the directory block for next admin block
func signDirBlockForAdminBlock(newdb *common.DirectoryBlock) error {
	if nodeMode == common.SERVER_NODE && dchain.NextDBHeight > 0 {
		//fmt.Printf("signDirBlockForAdminBlock: new dbBlock=%s\n", spew.Sdump(newdb.Header))
		//dbHeaderBytes, _ := newdb.Header.MarshalBinary()
		identityChainID := common.NewHash() // 0 ID for milestone 1 ????
		//sig := serverPrivKey.Sign(dbHeaderBytes)
		//achain.NextBlock.AddABEntry(common.NewDBSignatureEntry(identityChainID, sig))
		pub := common.PublicKey{
			Key: new([ed25519.PublicKeySize]byte),
		}
		sig := common.Signature{
			Pub: pub,
			Sig: new([ed25519.SignatureSize]byte),
		}
		achain.NextBlock.AddABEntry(common.NewDBSignatureEntry(identityChainID, sig))
	}
	return nil
}

// SignDirBlockForVote signs the directory block and broadcast it for vote for consensus
func SignDirBlockForVote(newdb *common.DirectoryBlock) error {
	if nodeMode == common.SERVER_NODE && dchain.NextDBHeight > 0 {
		//fmt.Printf("SignDirBlockForVote: new dbBlock=%s\n", spew.Sdump(newdb.Header))
		binaryDblock, err := newdb.MarshalBinary()
		if err != nil {
			return err
		}
		hash := common.Sha(binaryDblock)
		h := common.Hash{}
		h.SetBytes(hash.GetBytes())
		sig := serverPrivKey.Sign(h.GetBytes())
		msg := &wire.MsgDirBlockSig{
			DBHeight:     newdb.Header.DBHeight,
			DirBlockHash: h,
			Sig:          sig,
			SourceNodeID: localServer.nodeID,
		}
		outMsgQueue <- msg
		fmt.Println("my own: addDirBlockSig: ", spew.Sdump(msg))
		fMemPool.addDirBlockSig(msg)
	}
	return nil
}

// SignDirectoryBlock signs the directory block and broadcast it
func SignDirectoryBlock(newdb *common.DirectoryBlock) error {
	signDirBlockForAdminBlock(newdb)
	SignDirBlockForVote(newdb)
	return nil
}

// Place an anchor into btc
func placeAnchor(dblock *common.DirectoryBlock) error {
	if localServer.IsLeader() || localServer.isSingleServerMode() {
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

func backupKeyMapData() {
	chainIDMapBackup = make(map[string]*common.EChain)
	eCreditMapBackup = make(map[string]int32)
	commitChainMapBackup = make(map[string]*common.CommitChain, 0)
	commitEntryMapBackup = make(map[string]*common.CommitEntry, 0)

	// backup chainIDMap once a block is created, as a checkpoint.
	for k, v := range chainIDMap {
		chainIDMapBackup[k] = v
	}
	// backup eCreditMap
	for k, v := range eCreditMap {
		eCreditMapBackup[k] = v
	}
	for k, v := range commitChainMap {
		commitChainMapBackup[k] = v
	}
	for k, v := range commitEntryMap {
		commitEntryMapBackup[k] = v
	}
}

func resetLeaderState() {
	// reset eCreditMap & chainIDMap & processList
	fmt.Println("resetLeaderState: ")
	eCreditMap = eCreditMapBackup
	chainIDMap = chainIDMapBackup
	commitChainMap = commitChainMapBackup
	commitEntryMap = commitEntryMapBackup
	initProcessListMgr()

	// rebuild leader's process list
	var msg wire.FtmInternalMsg
	var hash *wire.ShaHash
	for i := 0; i < len(fMemPool.ackpool); i++ {
		if fMemPool.ackpool[i] == nil {
			continue
		}
		hash = fMemPool.ackpool[i].Affirmation	// never nil
		msg = fMemPool.pool[*hash]
		if msg == nil {
			if !fMemPool.ackpool[i].IsEomAck() {
				continue
			} else {
				msg = &wire.MsgInt_EOM{
					EOM_Type:         fMemPool.ackpool[i].Type,
					NextDBlockHeight: fMemPool.ackpool[i].Height,
				}
				hash = new(wire.ShaHash)
			}
		}
		fmt.Println("resetLeaderState: broadcast msg ", spew.Sdump(msg))
		outMsgQueue <- msg

		ack, _ := plMgr.AddToLeadersProcessList(msg, hash, fMemPool.ackpool[i].Type, 
			dchain.NextBlock.Header.Timestamp, fchain.NextBlock.GetCoinbaseTimestamp())
		fmt.Println("resetLeaderState: broadcast ack ", ack)
		outMsgQueue <- ack
		
		if fMemPool.ackpool[i].Type == wire.EndMinute10 {
			fmt.Println("resetLeaderState: stopped at EOM10")
			// should not get to this point
			break
		}
	}

	// start clock
	fmt.Println("resetLeaderState: @@@@ start BlockTimer for new current leader.")
	timer := &BlockTimer{
		nextDBlockHeight: dchain.NextDBHeight,
		inMsgQueue:       inMsgQueue,
	}
	go timer.StartBlockTimer()	

	// for missed EOMs, processLeaderEOM should take care them
	// relay stale messages in orphan pool and mempool in next round.
}

// check missing EOMs, in case of leader crash and a new leader emerges
func checkMissingLeaderEOMs(msgEom *wire.MsgInt_EOM) []*wire.MsgInt_EOM {
	var eom []byte
	var em []*wire.MsgInt_EOM
	for i := 1; i < int(msgEom.EOM_Type); i++ {
		eom = append(eom, byte(i))
	}
	for _, item := range plMgr.MyProcessList.GetPLItems() {
		if !item.Ack.IsEomAck() {
			continue
		}
		for j := 0; j < len(eom); j++ {
			if eom[j] == item.Ack.Type {
				fmt.Println("found missing EOM: ", eom[j])
				eom = append(eom[:j], eom[j+1:]...)
				break
			}
		}
	}
	if len(eom) > 0 {
		fmt.Println("missing leader EOMs: ", eom)
		for _, typ := range eom {
			m := &wire.MsgInt_EOM {
				EOM_Type: typ,
			}
			em = append(em, m)
		}
	}
	return em
}
