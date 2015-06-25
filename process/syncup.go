// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package process

import (
	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/util"
	"github.com/FactomProject/btcd/wire"
	"github.com/davecgh/go-spew/spew"
	"strconv"
	"time"
)

// processDirBlock validates dir block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processDirBlock(msg *wire.MsgDirBlock) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	blk, _ := db.FetchDBlockByHeight(msg.DBlk.Header.BlockHeight)
	if blk != nil {
		procLog.Info("DBlock already existing for height:" + string(msg.DBlk.Header.BlockHeight))
		return nil
	}

	msg.DBlk.IsSealed = true
	dchain.AddDBlockToDChain(msg.DBlk)

	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, strconv.Itoa(int(msg.DBlk.Header.BlockHeight))) // store in mempool with the height as the key

	procLog.Debugf("SyncUp: MsgDirBlock=%s\n", spew.Sdump(msg.DBlk))

	return nil
}

// processFBlock validates admin block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processFBlock(msg *wire.MsgFBlock) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	//Add it to mem pool before saving it in db
	h, _ := common.CreateHash(msg.SC)     // need to change it to MR??
	fMemPool.addBlockMsg(msg, h.String()) // stored in mem pool with the MR as the key

	return nil

}

// processABlock validates admin block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processABlock(msg *wire.MsgABlock) error {
	util.Trace()

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	//Add it to mem pool before saving it in db
	msg.ABlk.BuildABHash()
	fMemPool.addBlockMsg(msg, msg.ABlk.ABHash.String()) // store in mem pool with ABHash as key

	return nil
}

// procesFBlock validates entry credit block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func procesECBlock(msg *wire.MsgECBlock) error {
	util.Trace()

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	h, _ := common.CreateHash(msg.ECBlock)
	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, h.String())

	// for debugging??
	procLog.Debugf("SyncUp: MsgCBlock=%s\n", spew.Sdump(msg.ECBlock))

	return nil
}

// processEBlock validates entry block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processEBlock(msg *wire.MsgEBlock) error {
	util.Trace()

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	if msg.EBlk.Header.DBHeight >= dchain.NextBlockHeight || msg.EBlk.Header.DBHeight < 0 {
		return errors.New("MsgEBlock has an invalid DBHeight:" + strconv.Itoa(int(msg.EBlk.Header.DBHeight)))
	}

	//Add it to mem pool before saving it in db
	msg.EBlk.BuildMerkleRoot()
	fMemPool.addBlockMsg(msg, msg.EBlk.MerkleRoot.String()) // store it in mem pool with MR as the key

	// for debugging??
	procLog.Debugf("SyncUp: MsgEBlock=%s\n", spew.Sdump(msg.EBlk))

	return nil
}

// processEntry validates entry and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processEntry(msg *wire.MsgEntry) error {
	util.Trace()

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	// store the entry in mem pool
	h, _ := common.CreateHash(msg.Entry)
	fMemPool.addBlockMsg(msg, h.String()) // store it in mem pool with hash as the key

	procLog.Debugf("SyncUp: MsgEntry=%s\n", spew.Sdump(msg.Entry))

	return nil
}

// Validate the new blocks in mem pool and store them in db
func validateAndStoreBlocks(fMemPool *ftmMemPool, db database.Db, dchain *common.DChain, outCtlMsgQ chan wire.FtmInternalMsg) {
	var myDBHeight int64
	var sleeptime int
	var dblk *common.DirectoryBlock

	for true {
		dblk = nil
		_, myDBHeight, _ = db.FetchBlockHeightCache()

		adj := (len(dchain.Blocks) - int(myDBHeight))
		if adj <= 0 {
			adj = 1
		}
		// in milliseconds
		sleeptime = 100 + 1000/adj

		if len(dchain.Blocks) > int(myDBHeight+1) {
			dblk = dchain.Blocks[myDBHeight+1]
		}
		if dblk != nil {
			if validateBlocksFromMemPool(dblk, fMemPool, db) {
				err := storeBlocksFromMemPool(dblk, fMemPool, db)
				if err == nil {
					deleteBlocksFromMemPool(dblk, fMemPool)
				} else {
					panic("error in deleteBlocksFromMemPool.")
				}
			}
		} else {
			time.Sleep(time.Duration(sleeptime * 1000000)) // Nanoseconds for duration
					
			//send an internal msg to sync up with peers
			// ??
		}

	}

}

// Validate the new blocks in mem pool and store them in db
func validateBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool, db database.Db) bool {

	// Validate the genesis block
	if b.Header.BlockHeight == 0 {
		h, _ := common.CreateHash(b)
		if h.String() != common.GENESIS_DIR_BLOCK_HASH {
			// panic for milestone 1
			panic("Genesis dir block is not as expected: " + h.String())
		}
	}

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			if _, ok := fMemPool.blockpool[dbEntry.MerkleRoot.String()]; !ok {
				return false
			}
		case achain.ChainID.String():
			if _, ok := fMemPool.blockpool[dbEntry.MerkleRoot.String()]; !ok {
				return false
			}
		case fchain.ChainID.String():
			if _, ok := fMemPool.blockpool[dbEntry.MerkleRoot.String()]; !ok {
				return false
			}
		default:
			if msg, ok := fMemPool.blockpool[dbEntry.MerkleRoot.String()]; !ok {
				return false
			} else {
				eBlkMsg, _ := msg.(*wire.MsgEBlock)
				// validate every entry in EBlock
				for _, ebEntry := range eBlkMsg.EBlk.EBEntries {
					if _, foundInMemPool := fMemPool.blockpool[ebEntry.EntryHash.String()]; !foundInMemPool {
						// continue if the entry arleady exists in db
						entry, _ := db.FetchEntryByHash(ebEntry.EntryHash)
						if entry == nil {
							return false
						}
					}
				}
			}
		}
	}

	return true
}

// Validate the new blocks in mem pool and store them in db
// Need to make a batch insert in db in milestone 2
func storeBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool, db database.Db) error {

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			ecBlkMsg := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgECBlock)
			err := db.ProcessECBlockBatch(ecBlkMsg.ECBlock)
			if err != nil {
				return err
			}
			// needs to be improved??
			initializeECreditMap(ecBlkMsg.ECBlock)
			// for debugging
			exportECBlock(ecBlkMsg.ECBlock)
		case achain.ChainID.String():
			aBlkMsg := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgABlock)
			err := db.ProcessABlockBatch(aBlkMsg.ABlk)
			if err != nil {
				return err
			}
			// for debugging
			exportABlock(aBlkMsg.ABlk)			
		case fchain.ChainID.String():
			fBlkMsg := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgFBlock)
			err := db.ProcessFBlockBatch(fBlkMsg.SC)
			if err != nil {
				return err
			}
			// Initialize the Factoid State
        	err = common.FactoidState.AddTransactionBlock(fBlkMsg.SC)	
	        if err != nil { 
	            panic("Failed to rebuild factoid state: " +err.Error()); 
	        } 		
			
			// for debugging
			exportFctBlock(fBlkMsg.SC)			
		default:
			// handle Entry Block
			eBlkMsg, _ := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgEBlock)
			// store entry in db first
			for _, ebEntry := range eBlkMsg.EBlk.EBEntries {
				if msg, foundInMemPool := fMemPool.blockpool[ebEntry.EntryHash.String()]; foundInMemPool {
					err := db.InsertEntry(ebEntry.EntryHash, msg.(*wire.MsgEntry).Entry)
					if err != nil {
						return err
					}
				}
			}
			// Store Entry Block in db
			err := db.ProcessEBlockBatch(eBlkMsg.EBlk)
			if err != nil {
				return err
			}
			
			// create a chain in db if it's not existing
			chain := chainIDMap[eBlkMsg.EBlk.Header.ChainID.String()]
			if chain == nil {
				chain = new(common.EChain)
				chain.ChainID = eBlkMsg.EBlk.Header.ChainID
				if eBlkMsg.EBlk.Header.EBHeight == 0 {
					chain.FirstEntry, _ = db.FetchEntryByHash(eBlkMsg.EBlk.EBEntries[0].EntryHash)
				}
				db.InsertChain(chain)
				chainIDMap[chain.ChainID.String()] = chain
			} else if chain.FirstEntry == nil && eBlkMsg.EBlk.Header.EBHeight == 0 {
				chain.FirstEntry, _ = db.FetchEntryByHash(eBlkMsg.EBlk.EBEntries[0].EntryHash)
				db.InsertChain(chain)
			}
			
			// for debugging
			exportEBlock(eBlkMsg.EBlk)					
		}
	}

	// Store the dir block
	err := db.ProcessDBlockBatch(b)
	if err != nil {
		return err
	}
	
	// Update dir block height cache in db
	commonHash, _ := common.CreateHash(b)
	db.UpdateBlockHeightCache(b.Header.BlockHeight, commonHash)
		
	// for debugging
	exportDBlock(b)	

	return nil
}

// Validate the new blocks in mem pool and store them in db
func deleteBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool) error {

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			delete(fMemPool.blockpool, dbEntry.MerkleRoot.String())
		case achain.ChainID.String():
			delete(fMemPool.blockpool, dbEntry.MerkleRoot.String())
		case fchain.ChainID.String():
			delete(fMemPool.blockpool, dbEntry.MerkleRoot.String())
		default:
			eBlkMsg, _ := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgEBlock)
			for _, ebEntry := range eBlkMsg.EBlk.EBEntries {
				delete(fMemPool.blockpool, ebEntry.EntryHash.String())
			}
			delete(fMemPool.blockpool, dbEntry.MerkleRoot.String())
		}
	}
	delete(fMemPool.blockpool, strconv.Itoa(int(b.Header.BlockHeight)))

	return nil
}
