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

package process

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd/wire"
	"github.com/FactomProject/factoid/block"
	"github.com/davecgh/go-spew/spew"
)

var _ = (*block.FBlock)(nil)

var _ = util.Trace

var (
	db       database.Db        // database
	dchain   *common.DChain     //Directory Block Chain
	ecchain  *common.ECChain    //Entry Credit Chain
	achain   *common.AdminChain //Admin Chain
	fchain   *common.FctChain   // factoid Chain
	fchainID *common.Hash

	inMsgQueue  chan wire.FtmInternalMsg //incoming message queue for factom application messages
	outMsgQueue chan wire.FtmInternalMsg //outgoing message queue for factom application messages

	inCtlMsgQueue  chan wire.FtmInternalMsg //incoming message queue for factom control messages
	outCtlMsgQueue chan wire.FtmInternalMsg //outgoing message queue for factom control messages

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

	FactoshisPerCredit uint64 // .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents

	FactomdUser string
	FactomdPass string

	zeroHash = common.NewHash()
)

var (
	directoryBlockInSeconds int
	dataStorePath           string
	ldbpath                 string
	nodeMode                string
	devNet                  bool
	serverPrivKeyHex        string
	serverIndex             = common.NewServerIndexNumber()
)

// Get the configurations
func LoadConfigurations(cfg *util.FactomdConfig) {

	//setting the variables by the valued form the config file
	logLevel = cfg.Log.LogLevel
	dataStorePath = cfg.App.DataStorePath
	ldbpath = cfg.App.LdbPath
	directoryBlockInSeconds = cfg.App.DirectoryBlockInSeconds
	nodeMode = cfg.App.NodeMode
	serverPrivKeyHex = cfg.App.ServerPrivKey

	FactomdUser = cfg.Btc.RpcUser
	FactomdPass = cfg.Btc.RpcPass
}

// Initialize the processor
func initProcessor() {

	wire.Init()

	// init server private key or pub key
	initServerKeys()

	// init mem pools
	fMemPool = new(ftmMemPool)
	fMemPool.init_ftmMemPool()

	// init wire.FChainID
	wire.FChainID = common.NewHash()
	wire.FChainID.SetBytes(common.FACTOID_CHAINID)

	FactoshisPerCredit = 666666 // .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents

	// init Directory Block Chain
	initDChain()

	procLog.Info("Loaded ", dchain.NextDBHeight, " Directory blocks for chain: "+dchain.ChainID.String())

	// init Entry Credit Chain
	initECChain()
	procLog.Info("Loaded ", ecchain.NextBlockHeight, " Entry Credit blocks for chain: "+ecchain.ChainID.String())

	// init Admin Chain
	initAChain()
	procLog.Info("Loaded ", achain.NextBlockHeight, " Admin blocks for chain: "+achain.ChainID.String())

	initFctChain()
	//common.FactoidState.LoadState()
	procLog.Info("Loaded ", fchain.NextBlockHeight, " factoid blocks for chain: "+fchain.ChainID.String())

	anchor.InitAnchor(db)

	// build the Genesis blocks if the current height is 0
	if dchain.NextDBHeight == 0 && nodeMode == common.SERVER_NODE {
		buildGenesisBlocks()
	} else {
		// To be improved in milestone 2
		SignDirectoryBlock()
	}

	// init process list manager
	initProcessListMgr()

	// init Entry Chains
	initEChains()
	for _, chain := range chainIDMap {
		initEChainFromDB(chain)

		procLog.Info("Loaded ", chain.NextBlockHeight, " blocks for chain: "+chain.ChainID.String())
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
			nextDBlockHeight: dchain.NextDBHeight,
			inCtlMsgQueue:    inCtlMsgQueue,
		}
		go timer.StartBlockTimer()
	} else {
		// start the go routine to process the blocks and entries downloaded
		// from peers
		go validateAndStoreBlocks(fMemPool, db, dchain, outCtlMsgQueue)
	}

	// Process msg from the incoming queue one by one
	for {
		select {
		case msg := <-inMsgQ:

			if err := serveMsgRequest(msg); err != nil {
				procLog.Error(err)
			}

		case ctlMsg := <-inCtlMsgQueue:

			if err := serveMsgRequest(ctlMsg); err != nil {
				procLog.Error(err)
			}
		}
	}
}

// Serve the "fast lane" incoming control msg from inCtlMsgQueue
func serveCtlMsgRequest(msg wire.FtmInternalMsg) error {

	switch msg.Command() {
	case wire.CmdCommitChain:

	default:
		return errors.New("1 Message type unsupported:" + fmt.Sprintf("%+v", msg))
	}
	return nil

}

// Serve incoming msg from inMsgQueue
func serveMsgRequest(msg wire.FtmInternalMsg) error {

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

		if nodeMode == common.SERVER_NODE {
			msgEom, ok := msg.(*wire.MsgInt_EOM)
			if !ok {
				return errors.New("Error in build blocks:" + fmt.Sprintf("%+v", msg))
			}
			procLog.Infof("PROCESSOR: End of minute msg - wire.CmdInt_EOM:%+v\n", msg)

			common.FactoidState.EndOfPeriod(int(msgEom.EOM_Type))

			if msgEom.EOM_Type == wire.END_MINUTE_10 {

				// Process from Orphan pool before the end of process list
				processFromOrphanPool()

				// Pass the Entry Credit Exchange Rate into the Factoid component
				msgEom.EC_Exchange_Rate = FactoshisPerCredit
				plMgr.AddMyProcessListItem(msgEom, nil, wire.END_MINUTE_10)
				// Set exchange rate in the Factoid State
				common.FactoidState.SetFactoshisPerEC(FactoshisPerCredit)

				err := buildBlocks()
				if err != nil {
					return err
				}

			} else if wire.END_MINUTE_1 <= msgEom.EOM_Type && msgEom.EOM_Type < wire.END_MINUTE_10 {
				ack, err := plMgr.AddMyProcessListItem(msgEom, nil, msgEom.EOM_Type)
				if err != nil {
					return err
				}
				if ack.ChainID == nil {
					ack.ChainID = dchain.ChainID
				}
				// Broadcast the ack to the network if no errors
				//outMsgQueue <- ack
			}

			cp.CP.AddUpdate(
				"MinMark",  // tag
				"status",   // Category
				"Progress", // Title
				fmt.Sprintf("End of Minute %v\n", msgEom.EOM_Type)+ // Message
					fmt.Sprintf("Directory Block Height %v", dchain.NextDBHeight),
				0)
		}

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

	case wire.CmdFactoidTX:
		msgFactoidTX, ok := msg.(*wire.MsgFactoidTX)
		if ok && msgFactoidTX.IsValid() {
			t := msgFactoidTX.Transaction
			txnum := len(common.FactoidState.GetCurrentBlock().GetTransactions())
			if common.FactoidState.AddTransaction(txnum, t) == nil {
				if err := processBuyEntryCredit(msgFactoidTX); err != nil {
					return err
				}
			}
		} else {
			outMsgQueue <- msg
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

// processAcknowledgement validates the ack and adds it to processlist
func processAcknowledgement(msg *wire.MsgAcknowledgement) error {
	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	// Validate the signiture
	bytes, err := msg.GetBinaryForSignature()
	if err != nil {
		return err
	}
	if !serverPubKey.Verify(bytes, &msg.Signature) {
		return errors.New(fmt.Sprintf("Invalid signature in Ack = %s\n", spew.Sdump(msg)))
	}

	// Update the next block height in dchain
	if msg.Height > dchain.NextDBHeight {
		dchain.NextDBHeight = msg.Height
	}

	// Update the next block height in db
	if int64(msg.Height) > db.FetchNextBlockHeightCache() {
		db.UpdateNextBlockHeightCache(msg.Height)
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

		// 37 effectively removes the entry header.  1023 rounds up the credit calucation.
		cred := (len(bin) - 37 + 1023) / 1024

		if int32(c.Credits) < cred {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("Credit needs to paid first before an entry is revealed: %s", e.Hash().String())
		}

		// Add the msg to the Mem pool
		fMemPool.addMsg(msg, h)

		// Add to MyPL if Server Node
		if nodeMode == common.SERVER_NODE {
			if plMgr.IsMyPListExceedingLimit() {
				procLog.Warning("Exceeding MyProcessList size limit!")
				return fMemPool.addOrphanMsg(msg, h)
			}

			ack, err := plMgr.AddMyProcessListItem(msg, h,
				wire.ACK_REVEAL_ENTRY)
			if err != nil {
				return err
			} else {
				// Broadcast the ack to the network if no errors
				outMsgQueue <- ack
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
		newChain := common.NewEChain()
		newChain.ChainID = e.ChainID
		newChain.FirstEntry = e
		chainIDMap[e.ChainID.String()] = newChain

		cred := int32(binary.Size(bin)/1024 + 1 + 10)
		if int32(c.Credits) < cred {
			fMemPool.addOrphanMsg(msg, h)
			return fmt.Errorf("Credit needs to paid first before an entry is revealed: %s", e.Hash().String())
		}
		// Add the msg to the Mem pool
		fMemPool.addMsg(msg, h)

		// Add to MyPL if Server Node
		if nodeMode == common.SERVER_NODE {
			if plMgr.IsMyPListExceedingLimit() {
				procLog.Warning("Exceeding MyProcessList size limit!")
				return fMemPool.addOrphanMsg(msg, h)
			}
			ack, err := plMgr.AddMyProcessListItem(msg, h,
				wire.ACK_REVEAL_CHAIN)
			if err != nil {
				return err
			} else {
				// Broadcast the ack to the network if no errors
				outMsgQueue <- ack
			}
		}

		delete(commitChainMap, e.Hash().String())
		return nil
	} else {
		return fmt.Errorf("No commit for entry")
	}

	return nil
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

	// deduct the entry credits from the eCreditMap
	if eCreditMap[string(c.ECPubKey[:])] < int32(c.Credits) {
		return fmt.Errorf("Not enough credits for CommitEntry")
	}
	eCreditMap[string(c.ECPubKey[:])] -= int32(c.Credits)

	// add to the commitEntryMap
	commitEntryMap[c.EntryHash.String()] = c

	// Server: add to MyPL
	if nodeMode == common.SERVER_NODE {
		h, _ := msg.Sha()
		if plMgr.IsMyPListExceedingLimit() {
			procLog.Warning("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}

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

		if plMgr.IsMyPListExceedingLimit() {
			procLog.Warning("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}

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

// processBuyEntryCredit validates the MsgCommitChain and adds it to processlist
func processBuyEntryCredit(msg *wire.MsgFactoidTX) error {
	if nodeMode == common.SERVER_NODE {
		// Update the credit balance in memory
		for _, v := range msg.Transaction.GetECOutputs() {
			pub := new([32]byte)
			copy(pub[:], v.GetAddress().Bytes())

			cred := int32(v.GetAmount() / uint64(FactoshisPerCredit))

			eCreditMap[string(pub[:])] += cred

		}

		h, _ := msg.Sha()
		if plMgr.IsMyPListExceedingLimit() {
			procLog.Warning("Exceeding MyProcessList size limit!")
			return fMemPool.addOrphanMsg(msg, &h)
		}

		if _, err := plMgr.AddMyProcessListItem(msg, &h, wire.ACK_FACTOID_TX); err != nil {
			return err
		}
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
				procLog.Info("Error in processing orphan msgCommitChain:" + err.Error())
				continue
			}
			delete(fMemPool.orphans, k)

		case wire.CmdCommitEntry:
			msgCommitEntry, _ := msg.(*wire.MsgCommitEntry)
			err := processCommitEntry(msgCommitEntry)
			if err != nil {
				procLog.Info("Error in processing orphan msgCommitEntry:" + err.Error())
				continue
			}
			delete(fMemPool.orphans, k)

		case wire.CmdRevealEntry:
			msgRevealEntry, _ := msg.(*wire.MsgRevealEntry)
			err := processRevealEntry(msgRevealEntry)
			if err != nil {
				procLog.Info("Error in processing orphan msgRevealEntry:" + err.Error())
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
	// Allocate the first two dbentries for ECBlock and Factoid block
	dchain.AddDBEntry(&common.DBEntry{}) // AdminBlock
	dchain.AddDBEntry(&common.DBEntry{}) // ECBlock
	dchain.AddDBEntry(&common.DBEntry{}) // Factoid block

	// Entry Credit Chain
	cBlock := newEntryCreditBlock(ecchain)
	procLog.Debugf("buildGenesisBlocks: cBlock=%s\n", spew.Sdump(cBlock))
	dchain.AddECBlockToDBEntry(cBlock)
	exportECChain(ecchain)

	// Admin chain
	aBlock := newAdminBlock(achain)
	procLog.Debugf("buildGenesisBlocks: aBlock=%s\n", spew.Sdump(aBlock))
	dchain.AddABlockToDBEntry(aBlock)
	exportAChain(achain)

	// factoid Genesis Address
	//fchain.NextBlock = block.GetGenesisFBlock(0, FactoshisPerCredit, 10, 200000000000)
	fchain.NextBlock = block.GetGenesisFBlock()
	FBlock := newFactoidBlock(fchain)
	dchain.AddFBlockToDBEntry(FBlock)
	exportFctChain(fchain)

	// Directory Block chain
	procLog.Debug("in buildGenesisBlocks")
	dbBlock := newDirectoryBlock(dchain)

	// Check block hash if genesis block
	if dbBlock.DBHash.String() != common.GENESIS_DIR_BLOCK_HASH {

		procLog.Errorf("\nGenesis block hash expected: " + common.GENESIS_DIR_BLOCK_HASH +
			"\nGenesis block hash found:    " + dbBlock.DBHash.String() + "\n")
	}

	exportDChain(dchain)

	// place an anchor into btc
	placeAnchor(dbBlock)

	return nil
}

// build blocks from all process lists
func buildBlocks() error {

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
	exportECBlock(ecBlock)

	// Admin chain
	aBlock := newAdminBlock(achain)

	dchain.AddABlockToDBEntry(aBlock)
	exportABlock(aBlock)

	// Factoid chain
	fBlock := newFactoidBlock(fchain)

	dchain.AddFBlockToDBEntry(fBlock)
	exportFctBlock(fBlock)

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
		exportEBlock(eblock)
	}

	// Directory Block chain
	procLog.Debug("in buildBlocks")
	dbBlock := newDirectoryBlock(dchain)

	// Generate the inventory vector and relay it.
	binary, _ := dbBlock.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())
	outMsgQueue <- (&wire.MsgInt_DirBlock{hash})

	// Update dir block height cache in db
	db.UpdateBlockHeightCache(dbBlock.Header.DBHeight, commonHash)
	db.UpdateNextBlockHeightCache(dchain.NextDBHeight)

	exportDBlock(dbBlock)

	// re-initialize the process lit manager
	initProcessListMgr()

	// Initialize timer for the new dblock
	if nodeMode == common.SERVER_NODE {
		timer := &BlockTimer{
			nextDBlockHeight: dchain.NextDBHeight,
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
	block.Header.DBHeight = dchain.NextDBHeight
	block.Header.EntryCount = uint32(len(block.Body.EBEntries))

	chain.NextBlockHeight++
	var err error
	chain.NextBlock, err = common.MakeEBlock(chain, block)
	if err != nil {
		procLog.Debug("EntryBlock Error: " + err.Error())
		return nil
	}

	//Store the block in db
	db.ProcessEBlockBatch(block)
	procLog.Infof("EntryBlock: block" + strconv.FormatUint(uint64(block.Header.EBSequence), 10) + " created for chain: " + chain.ChainID.String())
	return block
}

// Seals the current open block, store it in db and create the next open block
func newEntryCreditBlock(chain *common.ECChain) *common.ECBlock {

	// acquire the last block
	block := chain.NextBlock

	if chain.NextBlockHeight != dchain.NextDBHeight {
		panic("Entry Credit Block height does not match Directory Block height:" + string(dchain.NextDBHeight))
	}

	block.BuildHeader()

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

	//Store the block in db
	db.ProcessECBlockBatch(block)
	procLog.Infof("EntryCreditBlock: block" + strconv.FormatUint(uint64(block.Header.DBHeight), 10) + " created for chain: " + chain.ChainID.String())

	return block
}

// Seals the current open block, store it in db and create the next open block
func newAdminBlock(chain *common.AdminChain) *common.AdminBlock {

	// acquire the last block
	block := chain.NextBlock

	if chain.NextBlockHeight != dchain.NextDBHeight {
		panic("Admin Block height does not match Directory Block height:" + string(dchain.NextDBHeight))
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

	// Create the block and add a new block for new coming entries
	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	chain.NextBlock, err = common.CreateAdminBlock(chain, block, 10)
	if err != nil {
		panic(err)
	}
	chain.BlockMutex.Unlock()

	//Store the block in db
	db.ProcessABlockBatch(block)
	procLog.Infof("Admin Block: block " + strconv.FormatUint(uint64(block.Header.DBHeight), 10) + " created for chain: " + chain.ChainID.String())

	return block
}

// Seals the current open block, store it in db and create the next open block
func newFactoidBlock(chain *common.FctChain) block.IFBlock {

	// acquire the last block
	currentBlock := chain.NextBlock

	if chain.NextBlockHeight != dchain.NextDBHeight {
		panic("Factoid Block height does not match Directory Block height:" + strconv.Itoa(int(dchain.NextDBHeight)))
	}

	chain.BlockMutex.Lock()
	chain.NextBlockHeight++
	common.FactoidState.SetFactoshisPerEC(FactoshisPerCredit)
	common.FactoidState.ProcessEndOfBlock2(chain.NextBlockHeight)
	chain.NextBlock = common.FactoidState.GetCurrentBlock()
	chain.BlockMutex.Unlock()

	//Store the block in db
	db.ProcessFBlockBatch(currentBlock)
	procLog.Infof("Factoid chain: block " + strconv.FormatUint(uint64(currentBlock.GetDBHeight()), 10) + " created for chain: " + chain.ChainID.String())

	return currentBlock
}

// Seals the current open block, store it in db and create the next open block
func newDirectoryBlock(chain *common.DChain) *common.DirectoryBlock {
	procLog.Debug("**** new Dir Block")
	// acquire the last block
	block := chain.NextBlock

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

	block.DBHash, _ = common.CreateHash(block)
	block.BuildKeyMerkleRoot()

	//Store the block in db
	db.ProcessDBlockBatch(block)

	// Initialize the dirBlockInfo obj in db
	db.InsertDirBlockInfo(common.NewDirBlockInfoFromDBlock(block))
	anchor.UpdateDirBlockInfoMap(common.NewDirBlockInfoFromDBlock(block))

	procLog.Info("DirectoryBlock: block" + strconv.FormatUint(uint64(block.Header.DBHeight), 10) + " created for directory block chain: " + chain.ChainID.String())

	// To be improved in milestone 2
	SignDirectoryBlock()

	return block
}

// Sign the directory block
func SignDirectoryBlock() error {
	// Only Servers can write the anchor to Bitcoin network
	if nodeMode == common.SERVER_NODE && dchain.NextDBHeight > 0 {
		// get the previous directory block from db
		dbBlock, _ := db.FetchDBlockByHeight(dchain.NextDBHeight - 1)
		dbHeaderBytes, _ := dbBlock.Header.MarshalBinary()
		identityChainID := common.NewHash() // 0 ID for milestone 1
		sig := serverPrivKey.Sign(dbHeaderBytes)
		achain.NextBlock.AddABEntry(common.NewDBSignatureEntry(identityChainID, sig))
	}
	return nil
}

// Place an anchor into btc
func placeAnchor(dbBlock *common.DirectoryBlock) error {
	// Only Servers can write the anchor to Bitcoin network
	if nodeMode == common.SERVER_NODE && dbBlock != nil {
		// todo: need to make anchor as a go routine, independent of factomd
		// same as blockmanager to btcd
		go anchor.SendRawTransactionToBTC(dbBlock.KeyMR, uint64(dbBlock.Header.DBHeight))
	}
	return nil
}
