// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package process

import (
	"errors"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
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

	blk, _ := db.FetchDBlockByHeight(msg.DBlk.Header.DBHeight)
	if blk != nil {
		procLog.Info("DBlock already existing for height:" + string(msg.DBlk.Header.DBHeight))
		return nil
	}

	msg.DBlk.IsSealed = true
	dchain.AddDBlockToDChain(msg.DBlk)

	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, strconv.Itoa(int(msg.DBlk.Header.DBHeight))) // store in mempool with the height as the key

	procLog.Debug("SyncUp: MsgDirBlock DBHeight=", msg.DBlk.Header.DBHeight)

	return nil
}

// processFBlock validates admin block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processFBlock(msg *wire.MsgFBlock) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	key, _ := msg.SC.GetHash().MarshalText()
	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, string(key)) // stored in mem pool with the MR as the key

	procLog.Debug("SyncUp: MsgFBlock DBHeight=", msg.SC.GetDBHeight())

	return nil

}

// processABlock validates admin block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processABlock(msg *wire.MsgABlock) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	//Add it to mem pool before saving it in db
	msg.ABlk.BuildABHash()
	fMemPool.addBlockMsg(msg, msg.ABlk.ABHash.String()) // store in mem pool with ABHash as key

	procLog.Debug("SyncUp: MsgABlock DBHeight=", msg.ABlk.Header.DBHeight)

	return nil
}

// procesFBlock validates entry credit block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func procesECBlock(msg *wire.MsgECBlock) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, msg.ECBlock.Header.Hash().String())

	// for debugging??
	procLog.Debug("SyncUp: MsgCBlock DBHeight=", msg.ECBlock.Header.DBHeight)

	return nil
}

// processEBlock validates entry block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processEBlock(msg *wire.MsgEBlock) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}
	/*
		if msg.EBlk.Header.DBHeight >= dchain.NextBlockHeight || msg.EBlk.Header.DBHeight < 0 {
			return errors.New("MsgEBlock has an invalid DBHeight:" + strconv.Itoa(int(msg.EBlk.Header.DBHeight)))
		}
	*/
	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, msg.EBlk.KeyMR().String()) // store it in mem pool with MR as the key

	// for debugging??
	procLog.Debug("SyncUp: MsgEBlock DBHeight=", msg.EBlk.Header.DBHeight)

	return nil
}

// processEntry validates entry and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processEntry(msg *wire.MsgEntry) error {

	// Error condiftion for Milestone 1
	if nodeMode == common.SERVER_NODE {
		return errors.New("Server received msg:" + msg.Command())
	}

	// store the entry in mem pool
	h := msg.Entry.Hash()
	fMemPool.addBlockMsg(msg, h.String()) // store it in mem pool with hash as the key

	procLog.Debug("SyncUp: MsgEntry hash=", msg.Entry.Hash())

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
			} else {
				time.Sleep(time.Duration(sleeptime * 1000000)) // Nanoseconds for duration
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
	if b.Header.DBHeight == 0 {
		h, _ := common.CreateHash(b)
		if h.String() != common.GENESIS_DIR_BLOCK_HASH {
			// panic for milestone 1
			panic("Genesis dir block is not as expected: " + h.String())
		}
	}

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			if _, ok := fMemPool.blockpool[dbEntry.KeyMR.String()]; !ok {
				return false
			}
		case achain.ChainID.String():
			if msg, ok := fMemPool.blockpool[dbEntry.KeyMR.String()]; !ok {
				return false
			} else {
				// validate signature of the previous dir block
				aBlkMsg, _ := msg.(*wire.MsgABlock)
				if !validateDBSignature(aBlkMsg.ABlk, dchain) {
					return false
				}
			}
		case fchain.ChainID.String():
			if _, ok := fMemPool.blockpool[dbEntry.KeyMR.String()]; !ok {
				return false
			}
		default:
			if msg, ok := fMemPool.blockpool[dbEntry.KeyMR.String()]; !ok {
				return false
			} else {
				eBlkMsg, _ := msg.(*wire.MsgEBlock)
				// validate every entry in EBlock
				for _, ebEntry := range eBlkMsg.EBlk.Body.EBEntries {
					if _, foundInMemPool := fMemPool.blockpool[ebEntry.String()]; !foundInMemPool {
						// continue if the entry arleady exists in db
						entry, _ := db.FetchEntryByHash(ebEntry)
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
			ecBlkMsg := fMemPool.blockpool[dbEntry.KeyMR.String()].(*wire.MsgECBlock)
			err := db.ProcessECBlockBatch(ecBlkMsg.ECBlock)
			if err != nil {
				return err
			}
			// needs to be improved??
			initializeECreditMap(ecBlkMsg.ECBlock)
			// for debugging
			exportECBlock(ecBlkMsg.ECBlock)
		case achain.ChainID.String():
			aBlkMsg := fMemPool.blockpool[dbEntry.KeyMR.String()].(*wire.MsgABlock)
			err := db.ProcessABlockBatch(aBlkMsg.ABlk)
			if err != nil {
				return err
			}
			// for debugging
			exportABlock(aBlkMsg.ABlk)
		case fchain.ChainID.String():
			fBlkMsg := fMemPool.blockpool[dbEntry.KeyMR.String()].(*wire.MsgFBlock)
			err := db.ProcessFBlockBatch(fBlkMsg.SC)
			if err != nil {
				return err
			}
			// Initialize the Factoid State
			err = common.FactoidState.AddTransactionBlock(fBlkMsg.SC)
			FactoshisPerCredit = fBlkMsg.SC.GetExchRate()
			if err != nil {
				return err
			}

			// for debugging
			exportFctBlock(fBlkMsg.SC)
		default:
			// handle Entry Block
			eBlkMsg, _ := fMemPool.blockpool[dbEntry.KeyMR.String()].(*wire.MsgEBlock)
			// store entry in db first
			for _, ebEntry := range eBlkMsg.EBlk.Body.EBEntries {
				if msg, foundInMemPool := fMemPool.blockpool[ebEntry.String()]; foundInMemPool {
					err := db.InsertEntry(msg.(*wire.MsgEntry).Entry)
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
				if eBlkMsg.EBlk.Header.EBSequence == 0 {
					chain.FirstEntry, _ = db.FetchEntryByHash(eBlkMsg.EBlk.Body.EBEntries[0])
				}
				db.InsertChain(chain)
				chainIDMap[chain.ChainID.String()] = chain
			} else if chain.FirstEntry == nil && eBlkMsg.EBlk.Header.EBSequence == 0 {
				chain.FirstEntry, _ = db.FetchEntryByHash(eBlkMsg.EBlk.Body.EBEntries[0])
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
	db.UpdateBlockHeightCache(b.Header.DBHeight, commonHash)

	// for debugging
	exportDBlock(b)

	return nil
}

// Validate the new blocks in mem pool and store them in db
func deleteBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool) error {

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			delete(fMemPool.blockpool, dbEntry.KeyMR.String())
		case achain.ChainID.String():
			delete(fMemPool.blockpool, dbEntry.KeyMR.String())
		case fchain.ChainID.String():
			delete(fMemPool.blockpool, dbEntry.KeyMR.String())
		default:
			eBlkMsg, _ := fMemPool.blockpool[dbEntry.KeyMR.String()].(*wire.MsgEBlock)
			for _, ebEntry := range eBlkMsg.EBlk.Body.EBEntries {
				delete(fMemPool.blockpool, ebEntry.String())
			}
			delete(fMemPool.blockpool, dbEntry.KeyMR.String())
		}
	}
	delete(fMemPool.blockpool, strconv.Itoa(int(b.Header.DBHeight)))

	return nil
}

func validateDBSignature(aBlock *common.AdminBlock, dchain *common.DChain) bool {

	dbSigEntry := aBlock.GetDBSignature()
	if dbSigEntry == nil {
		if aBlock.Header.DBHeight == 0 {
			return true
		} else {
			return false
		}
	} else {
		dbSig := dbSigEntry.(*common.DBSignatureEntry)
		if serverPubKey.String() != dbSig.PubKey.String() {
			return false
		} else {
			// obtain the previous directory block
			dblk := dchain.Blocks[aBlock.Header.DBHeight-1]
			if dblk == nil {
				return false
			} else {
				// validatet the signature
				bHeader, _ := dblk.Header.MarshalBinary()
				if !serverPubKey.Verify(bHeader, dbSig.PrevDBSig) {
					procLog.Infof("No valid signature found in Admin Block = %s\n", spew.Sdump(aBlock))
					return false
				}
			}
		}
	}

	return true
}
