// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package server

import (
	"bytes"
	//"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid/block"
	//cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/wire"
	"github.com/davecgh/go-spew/spew"
)

// processDirBlock validates dir block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processDirBlock(msg *wire.MsgDirBlock) error {
	fmt.Println("processDirBlock: height:", msg.DBlk.Header.DBHeight)
	blk, _ := db.FetchDBlockByHeight(msg.DBlk.Header.DBHeight)
	if blk != nil {
		/*
			procLog.Info("DBlock already exists for height:" + string(msg.DBlk.Header.DBHeight))
			cp.CP.AddUpdate(
				"DBOverlap",                                                          // tag
				"warning",                                                            // Category
				"Directory Block Overlap",                                            // Title
				"DBlock already exists for height:"+string(msg.DBlk.Header.DBHeight), // Message
				0) // Expire */
		return nil
	}

	msg.DBlk.IsSealed = true
	dchain.AddDBlockToDChain(msg.DBlk)

	//Add it to mem pool before saving it in db
	// msg.DBlk.BuildKeyMerkleRoot()
	// fMemPool.removeMissingMsg(msg.DBlk.KeyMR)
	
	fMemPool.addBlockMsg(msg, strconv.Itoa(int(msg.DBlk.Header.DBHeight))) // store in mempool with the height as the key
	/*
		//procLog.Debug("SyncUp: MsgDirBlock DBHeight=", msg.DBlk.Header.DBHeight)
		cp.CP.AddUpdate(
			"DBSyncUp", // tag
			"Status",   // Category
			"SyncUp:",  // Title
			"MsgDirBlock DBHeigth=:"+string(msg.DBlk.Header.DBHeight), // Message
			0) // Expire
	*/
	return nil
}

// processFBlock validates admin block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processFBlock(msg *wire.MsgFBlock) error {
	//key := hex.EncodeToString(msg.SC.GetHash().Bytes())
	//Add it to mem pool before saving it in db
	fMemPool.addBlockMsg(msg, msg.SC.GetHash().String()) //string(key)) // stored in mem pool with the MR as the key
	
	// h := common.NewHash()
	// h.SetBytes(msg.SC.GetHash().Bytes())
	// fMemPool.removeMissingMsg(h)
	
	fmt.Println("SyncUp: MsgFBlock DBHeight=", msg.SC.GetDBHeight())
	return nil

}

// processABlock validates admin block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processABlock(msg *wire.MsgABlock) error {
	//Add it to mem pool before saving it in db
	abHash, err := msg.ABlk.PartialHash()
	if err != nil {
		return err
	}
	fMemPool.addBlockMsg(msg, abHash.String()) // store in mem pool with ABHash as key
	// fMemPool.removeMissingMsg(abHash)
	fmt.Println("SyncUp: MsgABlock DBHeight=", msg.ABlk.Header.DBHeight)
	return nil
}

// procesFBlock validates entry credit block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func procesECBlock(msg *wire.MsgECBlock) error {
	//Add it to mem pool before saving it in db
	hash, err := msg.ECBlock.HeaderHash()
	if err != nil {
		return err
	}
	fMemPool.addBlockMsg(msg, hash.String())
	// fMemPool.removeMissingMsg(hash)
	fmt.Println("SyncUp: MsgCBlock DBHeight=", msg.ECBlock.Header.EBHeight)
	return nil
}

// processEBlock validates entry block and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processEBlock(msg *wire.MsgEBlock) error {
	//Add it to mem pool before saving it in db
	keyMR, err := msg.EBlk.KeyMR()
	if err != nil {
		return err
	}
	fMemPool.addBlockMsg(msg, keyMR.String()) // store it in mem pool with MR as the key
	// fMemPool.removeMissingMsg(keyMR)
	fmt.Println("SyncUp: MsgEBlock DBHeight=", msg.EBlk.Header.EBHeight)
	return nil
}

// processEntry validates entry and save it to factom db.
// similar to blockChain.BC_ProcessBlock
func processEntry(msg *wire.MsgEntry) error {
	// store the entry in mem pool
	h := msg.Entry.Hash()
	fMemPool.addBlockMsg(msg, h.String()) // store it in mem pool with hash as the key
	// fMemPool.removeMissingMsg(h)
	fmt.Println("SyncUp: MsgEntry hash=", msg.Entry.Hash())
	return nil
}

// Validate the new blocks in mem pool and store them in db
func validateAndStoreBlocks(fMemPool *ftmMemPool, db database.Db, dchain *common.DChain) {
	var myDBHeight int64
	var sleeptime int
	var dblk *common.DirectoryBlock

	procLog.Info("in validateAndStoreBlocks")
	for true {
		dblk = nil
		_, myDBHeight, _ = db.FetchBlockHeightCache()
		
		// remove blocks with myDBHeight from mempool if they exists
		deleteBlocksFromMemPoolByHeight(uint32(myDBHeight))

		adj := (len(dchain.Blocks) - int(myDBHeight))
		if adj <= 0 {
			adj = 1
		}
		// in milliseconds
		sleeptime = 100 + 1000/adj

		if len(dchain.Blocks) > int(myDBHeight+1) {
			dblk = dchain.Blocks[myDBHeight+1]
		}
		if dblk != nil && fMemPool.getDirBlock(uint32(myDBHeight+1)) != nil {
			if validateBlocksFromMemPool(dblk, fMemPool, db) {
				err := storeBlocksFromMemPool(dblk, fMemPool, db)
				if err == nil {
					deleteBlocksFromMemPool(dblk, fMemPool)
				} else {
					fmt.Println("error in storeBlocksFromMemPool. ", err.Error())
					//panic("error in deleteBlocksFromMemPool. " + err.Error())
				}
			} else {
				time.Sleep(time.Duration(sleeptime * 1000000)) // Nanoseconds for duration
			}
		} else {
			//requestMissingMsg(wire.InvTypeFactomDirBlock, zeroHash, uint32(myDBHeight+1))
			time.Sleep(time.Duration(sleeptime * 1000000)) // Nanoseconds for duration

			//TODO: send an internal msg to sync up with peers
		}

	}

}

// Validate the new blocks in mem pool and store them in db
func validateBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool, db database.Db) bool {
	procLog.Info("in validateBlocksFromMemPool. dir block height=", b.Header.DBHeight)

	// Validate the genesis block
	if b.Header.DBHeight == 0 {
		h, _ := common.CreateHash(b)
		if h.String() != common.GENESIS_DIR_BLOCK_HASH {
			// panic for milestone 1
			panic("\nGenesis block hash expected: " + common.GENESIS_DIR_BLOCK_HASH +
				"\nGenesis block hash found:    " + h.String() + "\n")
			//procLog.Errorf("Genesis dir block is not as expected: " + h.String())
		}
	}

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			if _, ok := fMemPool.FetchAndFoundFromBlockpool(dbEntry.KeyMR.String()); !ok {
				//requestMissingMsg(wire.InvTypeFactomEntryCreditBlock, dbEntry.KeyMR, b.Header.DBHeight)
				return false
			}
		case achain.ChainID.String():
			if msg, ok := fMemPool.FetchAndFoundFromBlockpool(dbEntry.KeyMR.String()); !ok {
				//requestMissingMsg(wire.InvTypeFactomAdminBlock, dbEntry.KeyMR, b.Header.DBHeight)
				return false
			} else {
				// validate signature of the previous dir block
				aBlkMsg, _ := msg.(*wire.MsgABlock)
				if !validateDBSignature(aBlkMsg.ABlk, dchain) {
					//return false
				}
			}
		case fchain.ChainID.String():
			if _, ok := fMemPool.FetchAndFoundFromBlockpool(dbEntry.KeyMR.String()); !ok {
				//requestMissingMsg(wire.InvTypeFactomFBlock, dbEntry.KeyMR, b.Header.DBHeight)
				return false
			}
		default:
			if msg, ok := fMemPool.FetchAndFoundFromBlockpool(dbEntry.KeyMR.String()); !ok {
				//requestMissingMsg(wire.InvTypeFactomEntryBlock, dbEntry.KeyMR, b.Header.DBHeight)
				return false
			} else {
				eBlkMsg, _ := msg.(*wire.MsgEBlock)
				// validate every entry in EBlock
				for _, ebEntry := range eBlkMsg.EBlk.Body.EBEntries {
					if _, foundInMemPool := fMemPool.FetchAndFoundFromBlockpool(ebEntry.String()); !foundInMemPool {
						if !bytes.Equal(ebEntry.Bytes()[:31], common.ZERO_HASH[:31]) {
							// continue if the entry arleady exists in db
							entry, _ := db.FetchEntryByHash(ebEntry)
							if entry == nil {
								//requestMissingMsg(wire.InvTypeFactomEntry, ebEntry, b.Header.DBHeight)
								return false
							}
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
	procLog.Info("in storeBlocksFromMemPool. dir block height=", b.Header.DBHeight)
	//db.StartBatch()

	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			ecBlkMsg := fMemPool.FetchFromBlockpool(dbEntry.KeyMR.String()).(*wire.MsgECBlock)
			//err := db.ProcessECBlockMultiBatch(ecBlkMsg.ECBlock)
			err := db.ProcessECBlockBatch(ecBlkMsg.ECBlock)
			if err != nil {
				return err
			}
			if ecchain.NextBlockHeight <= ecBlkMsg.ECBlock.Header.EBHeight {
				ecchain.NextBlockHeight = ecBlkMsg.ECBlock.Header.EBHeight + 1
			}
			// needs to be improved??
			initializeECreditMap(ecBlkMsg.ECBlock)
			// for debugging
			exportECBlock(ecBlkMsg.ECBlock)
		case achain.ChainID.String():
			aBlkMsg := fMemPool.FetchFromBlockpool(dbEntry.KeyMR.String()).(*wire.MsgABlock)
			//err := db.ProcessABlockMultiBatch(aBlkMsg.ABlk)
			err := db.ProcessABlockBatch(aBlkMsg.ABlk)
			if err != nil {
				return err
			}
			if achain.NextBlockHeight <= aBlkMsg.ABlk.Header.DBHeight {
				achain.NextBlockHeight = aBlkMsg.ABlk.Header.DBHeight + 1
			}
			// for debugging
			exportABlock(aBlkMsg.ABlk)
		case fchain.ChainID.String():
			fBlkMsg := fMemPool.FetchFromBlockpool(dbEntry.KeyMR.String()).(*wire.MsgFBlock)
			//err := db.ProcessFBlockMultiBatch(fBlkMsg.SC)
			err := db.ProcessFBlockBatch(fBlkMsg.SC)
			if err != nil {
				return err
			}
			// Initialize the Factoid State
			err = common.FactoidState.AddTransactionBlock(fBlkMsg.SC)
			FactoshisPerCredit = fBlkMsg.SC.GetExchRate()
			if err != nil {
				fmt.Println("err in FactoidState.AddTransactionBlock: ", spew.Sdump(fBlkMsg.SC))
				return err
			}
			if fchain.NextBlockHeight <= fBlkMsg.SC.GetDBHeight() {
				fchain.NextBlockHeight = fBlkMsg.SC.GetDBHeight() + 1
			}

			// in case of random fblock download, update fchain.NextBlock and factoid state
			common.FactoidState.ProcessEndOfBlock2(fchain.NextBlockHeight)
			fchain.NextBlock = common.FactoidState.GetCurrentBlock()
			t := block.GetCoinbase(common.FactoidState.GetTimeMilli())
			fchain.NextBlock.AddCoinbase(t)
			common.FactoidState.UpdateTransaction(t)
			doneSetFollowersCointbaseTimeStamp = false

			// for debugging
			exportFctBlock(fBlkMsg.SC)
		default:
			// handle Entry Block
			eBlkMsg, _ := fMemPool.FetchFromBlockpool(dbEntry.KeyMR.String()).(*wire.MsgEBlock)
			// store entry in db first
			for _, ebEntry := range eBlkMsg.EBlk.Body.EBEntries {
				if msg, foundInMemPool := fMemPool.FetchAndFoundFromBlockpool(ebEntry.String()); foundInMemPool {
					//err := db.InsertEntryMultiBatch(msg.(*wire.MsgEntry).Entry)
					err := db.InsertEntry(msg.(*wire.MsgEntry).Entry)
					if err != nil {
						return err
					}
				}
			}
			// Store Entry Block in db
			//err := db.ProcessEBlockMultiBatch(eBlkMsg.EBlk)
			err := db.ProcessEBlockBatch(eBlkMsg.EBlk)
			if err != nil {
				return err
			}

			// create a chain when it's the first block of the entry chain
			if eBlkMsg.EBlk.Header.EBSequence == 0 {
				chain := new(common.EChain)
				chain.ChainID = eBlkMsg.EBlk.Header.ChainID
				chain.FirstEntry, _ = db.FetchEntryByHash(eBlkMsg.EBlk.Body.EBEntries[0])
				if chain.FirstEntry == nil {
					return errors.New("First entry not found for chain:" + eBlkMsg.EBlk.Header.ChainID.String())
				}

				//err = db.InsertChainMultiBatch(chain)
				err = db.InsertChain(chain)
				if err != nil {
					return err
				}
				chainIDMap[chain.ChainID.String()] = chain
			}
			echain := chainIDMap[eBlkMsg.EBlk.Header.ChainID.String()]
			if echain.NextBlockHeight <= eBlkMsg.EBlk.Header.EBSequence {
				echain.NextBlockHeight = eBlkMsg.EBlk.Header.EBSequence + 1
			}			
			if echain.NextBlock == nil {
				echain.NextBlock = common.NewEBlock()
			}
			exportEBlock(eBlkMsg.EBlk)
		}
	}

	// Store the dir block
	//err := db.ProcessDBlockMultiBatch(b)
	err := db.ProcessDBlockBatch(b)
	if err != nil {
		return err
	}

	//err = db.EndBatch()
	//if err != nil {
	//return err
	//}

	// Update dir block height cache in db
	commonHash, _ := common.CreateHash(b)
	db.UpdateBlockHeightCache(b.Header.DBHeight, commonHash)

	// update the global network dirblock chain height
	if dchain.NextDBHeight <= b.Header.DBHeight {
		dchain.NextDBHeight = b.Header.DBHeight + 1
	}
	if plMgr.NextDBlockHeight <= b.Header.DBHeight {
		plMgr.NextDBlockHeight = b.Header.DBHeight + 1
		fmt.Println("syncup: update plMgr.NextDBlockHeight: ", plMgr.NextDBlockHeight)
	}

	// for debugging
	exportDBlock(b)
	
	// clean up mempool for client,
	if ClientOnly {
		fMemPool.cleanUpMemPoolClient(b.Header.DBHeight)
	}

	return nil
}

// Validate the new blocks in mem pool and store them in db
func deleteBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool) error {
	fmt.Println("in deleteBlocksFromMemPool. dir block height=", b.Header.DBHeight)
	for _, dbEntry := range b.DBEntries {
		switch dbEntry.ChainID.String() {
		case ecchain.ChainID.String():
			fMemPool.deleteBlockMsg(dbEntry.KeyMR.String())
		case achain.ChainID.String():
			fMemPool.deleteBlockMsg(dbEntry.KeyMR.String())
		case fchain.ChainID.String():
			fMemPool.deleteBlockMsg(dbEntry.KeyMR.String())
		default:
			eBlkMsg, _ := fMemPool.FetchFromBlockpool(dbEntry.KeyMR.String()).(*wire.MsgEBlock)
			for _, ebEntry := range eBlkMsg.EBlk.Body.EBEntries {
				fMemPool.deleteBlockMsg(ebEntry.String())
			}
			fMemPool.deleteBlockMsg(dbEntry.KeyMR.String())
		}
	}
	fMemPool.deleteBlockMsg(strconv.Itoa(int(b.Header.DBHeight)))

	return nil
}

func deleteBlocksFromMemPoolByHeight(h uint32) {
	dblock := fMemPool.getDirBlock(h)
	if dblock != nil {
		fmt.Println("in deleteBlocksFromMemPoolByHeight. dir block height=", h)
		deleteBlocksFromMemPool(dblock, fMemPool)
	}
}

func validateDBSignature(aBlock *common.AdminBlock, dchain *common.DChain) bool {
	procLog.Info("in validateDBSignature")

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
				if !serverPubKey.Verify(bHeader, (*[64]byte)(dbSig.PrevDBSig)) {
					procLog.Infof("No valid signature found in Admin Block = %s\n", spew.Sdump(aBlock))
					//return false
					return true
				}
			}
		}
	}

	return true
}

func requestMissingMsg(typ wire.InvType, hash *common.Hash, height uint32) {
	msg := fMemPool.addMissingMsg(typ, hash, height)
	if msg.TimesMissed > 100 && !msg.Requested {
		fmt.Println("Syncup: requestMissingMsg: ", spew.Sdump(msg))
		msg.Requested = true
		outMsgQueue <- msg.Msg
	}
}