package process

import (
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/btcd/wire"
	"strconv"
	"time"
)

// Validate the new blocks in mem pool and store them in db
func validateAndStoreBlocks(fMemPool *ftmMemPool, db database.Db, dchain *common.DChain, outCtlMsgQ chan wire.FtmInternalMsg) {
	var myDBHeight int64
	var sleeptime int
	var dblk *common.DirectoryBlock

	for true {
		dblk = nil
		_, myDBHeight, _ = db.FetchBlockHeightCache()

		// in milliseconds
		sleeptime = 100 + 1000/(len(dchain.Blocks)-int(myDBHeight))

		if len(dchain.Blocks) > int(myDBHeight+1) {
			dblk = dchain.Blocks[myDBHeight+1]
		}
		if dblk != nil {
			if validateBlocksFromMemPool(dblk, fMemPool, db) {
				err := storeBlocksFromMemPool(dblk, fMemPool, db)
				if err == nil {
					deleteBlocksFromMemPool(dblk, fMemPool)
					//to be removed ?? :
					exportDChain(dchain)
					exportAChain(achain)
					exportECChain(ecchain)
					exportSCChain(scchain)
				} else {
					panic("error in deleteBlocksFromMemPool.")
				}
			}
		} else {
			//send an internal msg to sync up with peers
		}

		time.Sleep(time.Duration(sleeptime * 1000000)) // Nanoseconds for duration
	}

}

// Validate the new blocks in mem pool and store them in db
func validateBlocksFromMemPool(b *common.DirectoryBlock, fMemPool *ftmMemPool, db database.Db) bool {

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
		case scchain.ChainID.String():
			if _, ok := fMemPool.blockpool[dbEntry.MerkleRoot.String()]; !ok {
				// ?? return false
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
		case achain.ChainID.String():
			aBlkMsg := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgABlock)
			err := db.ProcessABlockBatch(aBlkMsg.ABlk)
			if err != nil {
				return err
			}
		case scchain.ChainID.String():
			/*		fBlkMsg := fMemPool.blockpool[dbEntry.MerkleRoot.String()].(*wire.MsgFBlock)
					err := db.ProcessFBlockBatch(fBlkMsg.SC)
					if err != nil {
						return err
					}*/
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
		}
	}

	// Store the dir block
	err := db.ProcessDBlockBatch(b)
	if err != nil {
		return err
	}

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
		case scchain.ChainID.String():
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
