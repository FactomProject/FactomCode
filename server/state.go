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
	// "bytes"
	// "encoding/hex"
	// "errors"
	// "fmt"
	// "sort"
	// "strconv"
	// "sync"
	// "time"

	// "github.com/FactomProject/FactomCode/anchor"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	// "github.com/FactomProject/ed25519"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/FactomCode/wire"
	"github.com/FactomProject/factoid/block"
	// "github.com/davecgh/go-spew/spew"
)

var _ = (*block.FBlock)(nil)

type blockState struct {
	newDBlock  *common.DirectoryBlock
	newABlock  *common.AdminBlock
	newFBlock  block.IFBlock
	newECBlock *common.ECBlock
	newEBlocks []*common.EBlock

	commitChainMap map[string]*common.CommitChain
  commitChainMapBackup map[string]*common.CommitChain
	
	commitEntryMap map[string]*common.CommitEntry
	commitEntryMapBackup map[string]*common.CommitEntry

	chainIDMap       map[string]*common.EChain // ChainIDMap with chainID string([32]byte) as key
	chainIDMapBackup map[string]*common.EChain //previous block bakcup - ChainIDMap with chainID string([32]byte) as key

	eCreditMap       map[string]int32 // eCreditMap with public key string([32]byte) as key, credit balance as value
	eCreditMapBackup map[string]int32 // backup from previous block - eCreditMap with public key string([32]byte) as key, credit balance as value

	// FactoshisPerCredit is .001 / .15 * 100000000 (assuming a Factoid is .15 cents, entry credit = .1 cents
	FactoshisPerCredit                 uint64
	blockSyncing                       bool
	firstBlockHeight                   uint32 // the DBHeight of the first block being built by follower after sync up
	doneSetFollowersCointbaseTimeStamp bool
	doneSentCandidateMsg               bool
	leaderCrashed 										 bool
}

type state struct {
	localServer  *server
	db           database.Db // database
  
	inMsgQueue   chan wire.FtmInternalMsg
	outMsgQueue  chan wire.FtmInternalMsg

	dchain  *common.DChain     //Directory Block Chain
	ecchain *common.ECChain    //Entry Credit Chain
	achain  *common.AdminChain //Admin Chain
	fchain  *common.FctChain   // factoid Chain

	fMemPool *ftmMemPool
	plMgr    *consensus.ProcessListMgr

	//Server Private key and Public key for milestone 1
	serverPrivKey common.PrivateKey
	serverPubKey  common.PublicKey
	serverPrivKeyHex        string

	factomConfig *util.FactomdConfig
	directoryBlockInSeconds int
	dataStorePath           string
	ldbpath                 string
	nodeMode                string
	devNet                  bool
	// zeroHash                = common.NewHash()
	// serverIndex             = common.NewServerIndexNumber() //???
}

var (
  blockStateMap map[uint32]blockState
)
