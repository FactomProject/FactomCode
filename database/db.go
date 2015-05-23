package database

import (
	"github.com/FactomProject/FactomCode/common"
)

// AllShas is a special value that can be used as the final sha when requesting
// a range of shas by height to request them all.
const AllShas = int64(^uint64(0) >> 1)

// Db defines a generic interface that is used to request and insert data into db
type Db interface {
	// Close cleanly shuts down the database and syncs all data.
	Close() (err error)

	// RollbackClose discards the recent database changes to the previously
	// saved data at last Sync and closes the database.
	RollbackClose() (err error)

	// Sync verifies that the database is coherent on disk and no
	// outstanding transactions are in flight.
	Sync() (err error)

	// InsertEntry inserts an entry
	InsertEntry(entrySha *common.Hash, binaryEntry *[]byte, entry *common.Entry, chainID *[]byte) (err error)

	// FetchEntry gets an entry by hash from the database.
	FetchEntryByHash(entrySha *common.Hash) (entry *common.Entry, err error)

	// FetchEBEntriesFromQueue gets all of the ebentries that have not been processed
	//FetchEBEntriesFromQueue(chainID *[]byte, startTime *[]byte) (ebentries []*common.EBEntry, err error)

	// ProcessEBlockBatche inserts the EBlock and update all it's ebentries in DB
	ProcessEBlockBatch(eblock *common.EBlock) error

	// FetchDBEntriesFromQueue gets all of the dbentries that have not been processed
	//FetchDBEntriesFromQueue(startTime *[]byte) (dbentries []*common.DBEntry, err error)

	// InsertChain inserts the newly created chain into db
	InsertChain(chain *common.EChain) (err error)

	// FetchChainByHash gets a chain by chainID
	FetchChainByHash(chainID *common.Hash) (chain *common.EChain, err error)

	//FetchAllChains gets all of the chains
	FetchAllChains() (chains []common.EChain, err error)

	// FetchEntryInfoBranchByHash gets an EntryInfo obj
	//FetchEntryInfoByHash(entryHash *common.Hash) (entryInfo *common.EntryInfo, err error)

	// FetchEntryInfoBranchByHash gets an EntryInfoBranch obj
	// FetchEntryInfoBranchByHash(entryHash *common.Hash) (entryInfoBranch *common.EntryInfoBranch, err error)

	// FetchEntryBlock gets an entry by hash from the database.
	FetchEBlockByHash(eBlockHash *common.Hash) (eBlock *common.EBlock, err error)

	// FetchEBlockByMR gets an entry block by merkle root from the database.
	FetchEBlockByMR(eBMR *common.Hash) (eBlock *common.EBlock, err error)

	// FetchEBlockByHeight gets an entry block by height from the database.
	//FetchEBlockByHeight(chainID * common.Hash, eBlockHeight uint32) (eBlock *common.EBlock, err error)

	// FetchEBHashByMR gets an entry by hash from the database.
	FetchEBHashByMR(eBMR *common.Hash) (eBlockHash *common.Hash, err error)

	// FetchAllEBlocksByChain gets all of the blocks by chain id
	FetchAllEBlocksByChain(chainID *common.Hash) (eBlocks *[]common.EBlock, err error)

	// FetchDBlock gets an entry by hash from the database.
	FetchDBlockByHash(dBlockHash *common.Hash) (dBlock *common.DirectoryBlock, err error)

	// FetchDBlockByMR gets a directory block by merkle root from the database.
	FetchDBlockByMR(dBMR *common.Hash) (dBlock *common.DirectoryBlock, err error)

	// FetchDBHashByMR gets a DBHash by MR from the database.
	FetchDBHashByMR(dBMR *common.Hash) (dBlockHash *common.Hash, err error)

	// FetchDBBatchByHash gets an FBBatch obj
	FetchDirBlockInfoByHash(dbHash *common.Hash) (dirBlockInfo *common.DirBlockInfo, err error)

	// Insert the Directory Block meta data into db
	InsertDirBlockInfo(dirBlockInfo *common.DirBlockInfo) (err error)

	// FetchAllDirBlockInfo gets all of the dirBlockInfo
	FetchAllDirBlockInfo() (ddirBlockInfoMap map[string]*common.DirBlockInfo, err error)

	// FetchAllUnconfirmedDirBlockInfo gets all of the dirBlockInfos that have BTC Anchor confirmation
	//FetchAllUnconfirmedDirBlockInfo() (dBInfoSlice []common.DirBlockInfo, err error)
	FetchAllUnconfirmedDirBlockInfo() (dirBlockInfoMap map[string]*common.DirBlockInfo, err error)

	// ProcessDBlockBatche inserts the EBlock and update all it's ebentries in DB
	ProcessDBlockBatch(block *common.DirectoryBlock) error

	// FetchAllECBlocks gets all of the entry credit blocks
	FetchAllECBlocks() (cBlocks []common.ECBlock, err error)

	// FetchAllFBInfo gets all of the fbInfo
	FetchAllDBlocks() (fBlocks []common.DirectoryBlock, err error)

	// FetchDBlockByHeight gets an directory block by height from the database.
	FetchDBlockByHeight(dBlockHeight uint32) (dBlock *common.DirectoryBlock, err error) 
	
	// ProcessECBlockBatche inserts the ECBlock and update all it's ecbentries in DB
	ProcessECBlockBatch(block *common.ECBlock) (err error)

	// FetchECBlockByHash gets an Entry Credit block by hash from the database.
	FetchECBlockByHash(cBlockHash *common.Hash) (ecBlock *common.ECBlock, err error)

	// Initialize External ID map for explorer search
	InitializeExternalIDMap() (extIDMap map[string]bool, err error)

	// ProcessABlockBatch inserts the AdminBlock
	ProcessABlockBatch(block *common.AdminBlock) error

	// FetchABlockByHash gets an admin block by hash from the database.
	FetchABlockByHash(aBlockHash *common.Hash) (aBlock *common.AdminBlock, err error)

	// FetchAllABlocks gets all of the admin blocks
	FetchAllABlocks() (aBlocks []common.AdminBlock, err error)
}
