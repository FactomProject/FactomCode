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

	// FetchEBInfoByHash gets a chain by chainID
	FetchChainByHash(chainID *common.Hash) (chain *common.EChain, err error)

	// FetchChainByName gets a chain by chain name
	FetchChainByName(chainName [][]byte) (chain *common.EChain, err error)

	//FetchAllChains gets all of the chains
	FetchAllChains() (chains []common.EChain, err error)

	// FetchAllChainByName gets all of the chains under the path - name
	FetchAllChainsByName(chainName [][]byte) (chains *[]common.EChain, err error)

	// FetchEntryInfoBranchByHash gets an EntryInfo obj
	//FetchEntryInfoByHash(entryHash *common.Hash) (entryInfo *common.EntryInfo, err error)

	// FetchEntryInfoBranchByHash gets an EntryInfoBranch obj
	// FetchEntryInfoBranchByHash(entryHash *common.Hash) (entryInfoBranch *common.EntryInfoBranch, err error)

	// FetchEntryBlock gets an entry by hash from the database.
	FetchEBlockByHash(eBlockHash *common.Hash) (eBlock *common.EBlock, err error)

	// FetchEBlockByMR gets an entry block by merkle root from the database.
	FetchEBlockByMR(eBMR *common.Hash) (eBlock *common.EBlock, err error)

	// FetchEBlockByHeight gets an entry block by height from the database.
	FetchEBlockByHeight(chainID * common.Hash, eBlockHeight uint64) (eBlock *common.EBlock, err error)

	// FetchEBHashByMR gets an entry by hash from the database.
	FetchEBHashByMR(eBMR *common.Hash) (eBlockHash *common.Hash, err error)

	// FetchEBInfoByHash gets an EBInfo obj
	FetchEBInfoByHash(ebHash *common.Hash) (ebInfo *common.EBInfo, err error)

	// FetchAllEBlocksByChain gets all of the blocks by chain id
	FetchAllEBlocksByChain(chainID *common.Hash) (eBlocks *[]common.EBlock, err error)

	// FetchAllEBInfosByChain gets all of the entry block infos by chain id
	FetchAllEBInfosByChain(chainID *common.Hash) (eBInfos *[]common.EBInfo, err error)

	// FetchDBlock gets an entry by hash from the database.
	FetchDBlockByHash(dBlockHash *common.Hash) (dBlock *common.DirectoryBlock, err error)

	// FetchDBBatchByHash gets an FBBatch obj
	FetchDBInfoByHash(dbHash *common.Hash) (dbInfo *common.DBInfo, err error)

	// Insert the Directory Block meta data into db
	InsertDBInfo(dbInfo common.DBInfo) (err error)

	// ProcessDBlockBatche inserts the EBlock and update all it's ebentries in DB
	ProcessDBlockBatch(block *common.DirectoryBlock) error

	// FetchAllCBlocks gets all of the entry credit blocks
	FetchAllCBlocks() (cBlocks []common.CBlock, err error)

	// FetchAllFBInfo gets all of the fbInfo
	FetchAllDBlocks() (fBlocks []common.DirectoryBlock, err error)
	
	// FetchDBlockByHeight gets an directory block by height from the database.
	FetchDBlockByHeight(dBlockHeight uint64) (dBlock *common.DirectoryBlock, err error) 
	
	// ProcessCBlockBatche inserts the CBlock and update all it's cbentries in DB
	ProcessCBlockBatch(block *common.CBlock) (err error)

	// FetchCBlockByHash gets an Entry Credit block by hash from the database.
	FetchCBlockByHash(cBlockHash *common.Hash) (cBlock *common.CBlock, err error)

	// Initialize External ID map for explorer search
	InitializeExternalIDMap() (extIDMap map[string]bool, err error)

	/*
		// ProcessFBlockBatche inserts the FBlock
		ProcessFBlockBatch(block *factoid.FBlock) error

		// FetchFBInfoByHash gets an FBInfo obj
		//FetchFBInfoByHash(fbHash *common.Hash) (fbInfo *common.FBInfo, err error)

		// FetchAllFBlocks gets all of the factoid blocks
		FetchAllFBlocks() (fBlocks []factoid.FBlock, err error)
	*/
}
