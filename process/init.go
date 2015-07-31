// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package process

import (
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/factomlog"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd/wire"
	fct "github.com/FactomProject/factoid"
	"github.com/davecgh/go-spew/spew"
	"sort"
	"strconv"
)

// Initialize Directory Block Chain from database
func initDChain() {
	dchain = new(common.DChain)

	//Initialize the Directory Block Chain ID
	dchain.ChainID = new(common.Hash)
	barray := common.D_CHAINID
	dchain.ChainID.SetBytes(barray)

	// get all dBlocks from db
	dBlocks, _ := db.FetchAllDBlocks()
	sort.Sort(util.ByDBlockIDAccending(dBlocks))

	dchain.Blocks = make([]*common.DirectoryBlock, len(dBlocks), len(dBlocks)+1)

	for i := 0; i < len(dBlocks); i = i + 1 {
		if dBlocks[i].Header.DBHeight != uint32(i) {
			panic("Error in initializing dChain:" + dchain.ChainID.String())
		}
		dBlocks[i].Chain = dchain
		dBlocks[i].IsSealed = true
		dBlocks[i].IsSavedInDB = true
		dchain.Blocks[i] = &dBlocks[i]
	}

	// double check the block ids
	for i := 0; i < len(dchain.Blocks); i = i + 1 {
		if uint32(i) != dchain.Blocks[i].Header.DBHeight {
			panic(errors.New("BlockID does not equal index for chain:" + dchain.ChainID.String() + " block:" + fmt.Sprintf("%v", dchain.Blocks[i].Header.DBHeight)))
		}
	}

	//Create an empty block and append to the chain
	if len(dchain.Blocks) == 0 {
		dchain.NextDBHeight = 0
		dchain.NextBlock, _ = common.CreateDBlock(dchain, nil, 10)
	} else {
		dchain.NextDBHeight = uint32(len(dchain.Blocks))
		dchain.NextBlock, _ = common.CreateDBlock(dchain, dchain.Blocks[len(dchain.Blocks)-1], 10)
		// Update dir block height cache in db
		db.UpdateBlockHeightCache(dchain.NextDBHeight-1, dchain.NextBlock.Header.PrevFullHash)
	}

	exportDChain(dchain)

	//Double check the sealed flag
	if dchain.NextBlock.IsSealed == true {
		panic("dchain.Blocks[dchain.NextBlockID].IsSealed for chain:" + dchain.ChainID.String())
	}

}

// Initialize Entry Credit Block Chain from database
func initECChain() {

	eCreditMap = make(map[string]int32)

	//Initialize the Entry Credit Chain ID
	ecchain = common.NewECChain()

	// get all ecBlocks from db
	ecBlocks, _ := db.FetchAllECBlocks()
	sort.Sort(util.ByECBlockIDAccending(ecBlocks))

	for i, v := range ecBlocks {
		if v.Header.DBHeight != uint32(i) {
			panic("Error in initializing dChain:" + ecchain.ChainID.String() + " DBHeight:" + strconv.Itoa(int(v.Header.DBHeight)) + " i:" + strconv.Itoa(i))
		}

		// Calculate the EC balance for each account
		initializeECreditMap(&v)
	}

	//Create an empty block and append to the chain
	if len(ecBlocks) == 0 || dchain.NextDBHeight == 0 {
		ecchain.NextBlockHeight = 0
		ecchain.NextBlock = common.NewECBlock()
	} else {
		// Entry Credit Chain should have the same height as the dir chain
		ecchain.NextBlockHeight = dchain.NextDBHeight
		ecchain.NextBlock = common.NextECBlock(&ecBlocks[ecchain.NextBlockHeight-1])
	}

	// create a backup copy before processing entries
	copyCreditMap(eCreditMap, eCreditMapBackup)
	exportECChain(ecchain)

	// ONly for debugging
	if procLog.Level() > factomlog.Info {
		printCreditMap()
	}

}

// Initialize Admin Block Chain from database
func initAChain() {

	//Initialize the Admin Chain ID
	achain = new(common.AdminChain)
	achain.ChainID = new(common.Hash)
	achain.ChainID.SetBytes(common.ADMIN_CHAINID)

	// get all aBlocks from db
	aBlocks, _ := db.FetchAllABlocks()
	sort.Sort(util.ByABlockIDAccending(aBlocks))

	// double check the block ids
	for i := 0; i < len(aBlocks); i = i + 1 {
		if uint32(i) != aBlocks[i].Header.DBHeight {
			panic(errors.New("BlockID does not equal index for chain:" + achain.ChainID.String() + " block:" + fmt.Sprintf("%v", aBlocks[i].Header.DBHeight)))
		}
		if !validateDBSignature(&aBlocks[i], dchain) {
			panic(errors.New("No valid signature found in Admin Block = " + fmt.Sprintf("%s\n", spew.Sdump(aBlocks[i]))))
		}
	}

	//Create an empty block and append to the chain
	if len(aBlocks) == 0 || dchain.NextDBHeight == 0 {
		achain.NextBlockHeight = 0
		achain.NextBlock, _ = common.CreateAdminBlock(achain, nil, 10)

	} else {
		// Entry Credit Chain should have the same height as the dir chain
		achain.NextBlockHeight = dchain.NextDBHeight
		achain.NextBlock, _ = common.CreateAdminBlock(achain, &aBlocks[achain.NextBlockHeight-1], 10)
	}

	exportAChain(achain)

}

// Initialize Factoid Block Chain from database
func initFctChain() {

	//Initialize the Admin Chain ID
	fchain = new(common.FctChain)
	fchain.ChainID = new(common.Hash)
	fchain.ChainID.SetBytes(fct.FACTOID_CHAINID)

	// get all aBlocks from db
	fBlocks, _ := db.FetchAllFBlocks()
	sort.Sort(util.ByFBlockIDAccending(fBlocks))

	// double check the block ids
	for i := 0; i < len(fBlocks); i = i + 1 {
		if uint32(i) != fBlocks[i].GetDBHeight() {
			panic(errors.New("BlockID does not equal index for chain:" +
				fchain.ChainID.String() + " block:" +
				fmt.Sprintf("%v", fBlocks[i].GetDBHeight())))
		} else {
			// initialize the FactoidState in sequence
			err := common.FactoidState.AddTransactionBlock(fBlocks[i])
			if err != nil {
				panic("Failed to rebuild factoid state: " + err.Error())
			}
		}
	}

	//Create an empty block and append to the chain
	if len(fBlocks) == 0 || dchain.NextDBHeight == 0 {
		fchain.NextBlockHeight = 0
		fchain.NextBlock = getGenesisFBlock()
		fmt.Println(fchain.NextBlock)
	} else {
		fchain.NextBlockHeight = dchain.NextDBHeight
	}
	common.FactoidState.ProcessEndOfBlock2(dchain.NextDBHeight)
	fchain.NextBlock = common.FactoidState.GetCurrentBlock()

	exportFctChain(fchain)

}

// Initialize Entry Block Chains from database
func initEChains() {

	chainIDMap = make(map[string]*common.EChain)

	chains, err := db.FetchAllChains()

	if err != nil {
		panic(err)
	}

	for _, chain := range chains {
		var newChain = chain
		chainIDMap[newChain.ChainID.String()] = newChain
		exportEChain(chain)
	}

}

// Re-calculate Entry Credit Balance Map with a new Entry Credit Block
func initializeECreditMap(block *common.ECBlock) {
	for _, entry := range block.Body.Entries {
		// Only process: ECIDChainCommit, ECIDEntryCommit, ECIDBalanceIncrease
		switch entry.ECID() {
		case common.ECIDChainCommit:
			e := entry.(*common.CommitChain)
			eCreditMap[string(e.ECPubKey[:])] += int32(e.Credits)
			common.FactoidState.UpdateECBalance(fct.NewAddress(e.ECPubKey[:]), int64(e.Credits))
		case common.ECIDEntryCommit:
			e := entry.(*common.CommitEntry)
			eCreditMap[string(e.ECPubKey[:])] += int32(e.Credits)
			common.FactoidState.UpdateECBalance(fct.NewAddress(e.ECPubKey[:]), int64(e.Credits))
		case common.ECIDBalanceIncrease:
			e := entry.(*common.IncreaseBalance)
			eCreditMap[string(e.ECPubKey[:])] += int32(e.NumEC)
			// Don't add the Increases to Factoid state, the Factoid processing will do that.
		case common.ECIDServerIndexNumber:
		case common.ECIDMinuteNumber:
		default:
			panic("Unknow entry type:" + string(entry.ECID()) + " for ECBlock:" + strconv.FormatUint(uint64(block.Header.DBHeight), 10))
		}
	}
}

// Initialize server private key and server public key for milestone 1
func initServerKeys() {
	if nodeMode == common.SERVER_NODE {
		var err error
		serverPrivKey, err = common.NewPrivateKeyFromHex(serverPrivKeyHex)
		if err != nil {
			panic("Cannot parse Server Private Key from configuration file: " + err.Error())
		}
	}

	serverPubKey = common.PubKeyFromString(common.SERVER_PUB_KEY)

}

// Initialize the process list manager with the proper dir block height
func initProcessListMgr() {
	plMgr = consensus.NewProcessListMgr(dchain.NextDBHeight, 1, 10, serverPrivKey)

}

// Initialize the entry chains in memory from db
func initEChainFromDB(chain *common.EChain) {

	eBlocks, _ := db.FetchAllEBlocksByChain(chain.ChainID)
	sort.Sort(util.ByEBlockIDAccending(*eBlocks))

	for i := 0; i < len(*eBlocks); i = i + 1 {
		if uint32(i) != (*eBlocks)[i].Header.EBSequence {
			panic(errors.New("BlockID does not equal index for chain:" + chain.ChainID.String() + " block:" + fmt.Sprintf("%v", (*eBlocks)[i].Header.EBSequence)))
		}
	}

	if len(*eBlocks) == 0 {
		chain.NextBlockHeight = 0
		chain.NextBlock = common.MakeEBlock(chain, nil)
	} else {
		chain.NextBlockHeight = uint32(len(*eBlocks))
		chain.NextBlock = common.MakeEBlock(chain, &(*eBlocks)[len(*eBlocks)-1])
	}

	// Initialize chain with the first entry (Name and rules) for non-server mode
	if nodeMode != common.SERVER_NODE && chain.FirstEntry == nil && len(*eBlocks) > 0 {
		chain.FirstEntry, _ = db.FetchEntryByHash((*eBlocks)[0].Body.EBEntries[0])
		if chain.FirstEntry != nil {
			db.InsertChain(chain)
		}
	}

}

// Validate dir chain from genesis block
func validateDChain(c *common.DChain) error {

	if nodeMode != common.SERVER_NODE && len(c.Blocks) == 0 {
		return nil
	}

	if uint32(len(c.Blocks)) != c.NextDBHeight {
		return errors.New("Dir chain has an un-expected Next Block ID: " + strconv.Itoa(int(c.NextDBHeight)))
	}

	//prevMR and prevBlkHash are used to validate against the block next in the chain
	prevMR, prevBlkHash, err := validateDBlock(c, c.Blocks[0])
	if err != nil {
		return err
	}

	//validate the genesis block
	//prevBlkHash is the block hash for c.Blocks[0]
	if prevBlkHash == nil || prevBlkHash.String() != common.GENESIS_DIR_BLOCK_HASH {

		procLog.Errorf("Genesis Block wasn't as expected:\n" +
			"    Expected: " + common.GENESIS_DIR_BLOCK_HASH + "\n" +
			"    Found:    " + prevBlkHash.String())

		str := fmt.Sprintf("<pre>" +
			"Expected: " + common.GENESIS_DIR_BLOCK_HASH + "<br>" +
			"Found:    " + prevBlkHash.String() + "</pre><br><br>")
		cp.CP.AddUpdate(
			"GenHash",                    // tag
			"warning",                    // Category
			"Genesis Hash doesn't match", // Title
			str, // Message
			0)

	}

	for i := 1; i < len(c.Blocks); i++ {
		if !prevBlkHash.IsSameAs(c.Blocks[i].Header.PrevFullHash) {
			return errors.New("Previous block hash not matching for Dir block: " + strconv.Itoa(i))
		}
		if !prevMR.IsSameAs(c.Blocks[i].Header.PrevKeyMR) {
			return errors.New("Previous merkle root not matching for Dir block: " + strconv.Itoa(i))
		}
		mr, dblkHash, err := validateDBlock(c, c.Blocks[i])
		if err != nil {
			c.Blocks[i].IsValidated = false
			return err
		}

		prevMR = mr
		prevBlkHash = dblkHash
		c.Blocks[i].IsValidated = true
	}

	return nil
}

// Validate a dir block
func validateDBlock(c *common.DChain, b *common.DirectoryBlock) (merkleRoot *common.Hash, dbHash *common.Hash, err error) {

	bodyMR, err := b.BuildBodyMR()
	if err != nil {
		return nil, nil, err
	}

	if !b.Header.BodyMR.IsSameAs(bodyMR) {
		return nil, nil, errors.New("Invalid body MR for dir block: " + string(b.Header.DBHeight))
	}

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			err := validateCBlockByMR(dbEntry.KeyMR)
			if err != nil {
				return nil, nil, err
			}
		case achain.ChainID.String():
			err := validateABlockByMR(dbEntry.KeyMR)
			if err != nil {
				return nil, nil, err
			}
		case wire.FChainID.String():
			err := validateFBlockByMR(dbEntry.KeyMR)
			if err != nil {
				return nil, nil, err
			}
		default:
			err := validateEBlockByMR(dbEntry.ChainID, dbEntry.KeyMR)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	b.DBHash, _ = common.CreateHash(b)
	b.BuildKeyMerkleRoot()

	return b.KeyMR, b.DBHash, nil
}

// Validate Entry Credit Block by merkle root
func validateCBlockByMR(mr *common.Hash) error {
	cb, _ := db.FetchECBlockByHash(mr)

	if cb == nil {
		return errors.New("Entry Credit block not found in db for merkle root: " + mr.String())
	}

	return nil
}

// Validate Admin Block by merkle root
func validateABlockByMR(mr *common.Hash) error {
	b, _ := db.FetchABlockByHash(mr)

	if b == nil {
		return errors.New("Admin block not found in db for merkle root: " + mr.String())
	}

	return nil
}

// Validate FBlock by merkle root
func validateFBlockByMR(mr *common.Hash) error {
	b, _ := db.FetchFBlockByHash(mr)

	if b == nil {
		return errors.New("Simple Coin block not found in db for merkle root: " + mr.String())
	}

	return nil
}

// Validate Entry Block by merkle root
func validateEBlockByMR(cid *common.Hash, mr *common.Hash) error {

	eb, _ := db.FetchEBlockByMR(mr)

	if eb == nil {
		return errors.New("Entry block not found in db for merkle root: " + mr.String())
	}

	if !mr.IsSameAs(eb.KeyMR()) {
		return errors.New("Entry block's merkle root does not match with: " + mr.String())
	}

	for _, ebEntry := range eb.Body.EBEntries {
		entry, _ := db.FetchEntryByHash(ebEntry)
		if entry == nil {
			return errors.New("Entry not found in db for entry hash: " + ebEntry.String())
		}
	}

	return nil
}
