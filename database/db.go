package database

import (
	"github.com/FactomProject/FactomCode/factomchain/factoid"
	"github.com/FactomProject/FactomCode/notaryapi"
)

// AllShas is a special value that can be used as the final sha when requesting
// a range of shas by height to request them all.
const AllShas = int64(^uint64(0) >> 1)

// Db defines a generic interface that is used to request and insert data into db
type Db interface {
	// Close cleanly shuts down the database and syncs all data.
	Close() (err error)

	// InsertEntry inserts an entry and put it on a process queue
	InsertEntryAndQueue(entrySha *notaryapi.Hash, binaryEntry *[]byte, entry *notaryapi.Entry, chainID *[]byte) (err error)

	// FetchEntry gets an entry by hash from the database.
	FetchEntryByHash(entrySha *notaryapi.Hash) (entry *notaryapi.Entry, err error)

	// FetchEBEntriesFromQueue gets all of the ebentries that have not been processed
	FetchEBEntriesFromQueue(chainID *[]byte, startTime *[]byte) (ebentries []*notaryapi.EBEntry, err error)

	// ProcessEBlockBatche inserts the EBlock and update all it's ebentries in DB
	ProcessEBlockBatch(eblock *notaryapi.EBlock) error

	// FetchDBEntriesFromQueue gets all of the dbentries that have not been processed
	FetchDBEntriesFromQueue(startTime *[]byte) (dbentries []*notaryapi.DBEntry, err error)

	// ProcessDBlockBatche inserts the EBlock and update all it's ebentries in DB
	ProcessDBlockBatch(block *notaryapi.DBlock) error

	// InsertChain inserts the newly created chain into db
	InsertChain(chain *notaryapi.EChain) (err error)

	// FetchEBInfoByHash gets a chain by chainID
	FetchChainByHash(chainID *notaryapi.Hash) (chain *notaryapi.EChain, err error)

	// FetchChainByName gets a chain by chain name
	FetchChainByName(chainName [][]byte) (chain *notaryapi.EChain, err error)

	// RollbackClose discards the recent database changes to the previously
	// saved data at last Sync and closes the database.
	RollbackClose() (err error)

	// Sync verifies that the database is coherent on disk and no
	// outstanding transactions are in flight.
	Sync() (err error)

	// FetchAllChainByName gets all of the chains under the path - name
	FetchAllChainsByName(chainName [][]byte) (chains *[]notaryapi.EChain, err error)

	// Insert the Factom Block meta data into db
	//InsertFBInfo(fbHash *notaryapi.Hash, fbInfo *notaryapi.FBInfo) (err error)

	// Insert the Directory Block meta data into db
	InsertDBBatch(fbBatch *notaryapi.DBBatch) (err error)

	// FetchEBInfoByHash gets an EBInfo obj
	FetchEBInfoByHash(ebHash *notaryapi.Hash) (ebInfo *notaryapi.EBInfo, err error)

	// FetchFBInfoByHash gets an FBInfo obj
	//FetchFBInfoByHash(fbHash *notaryapi.Hash) (fbInfo *notaryapi.FBInfo, err error)

	// FetchDBBatchByHash gets an FBBatch obj
	FetchDBBatchByHash(fbHash *notaryapi.Hash) (fbBatch *notaryapi.DBBatch, err error)

	// FetchEntryInfoBranchByHash gets an EntryInfo obj
	FetchEntryInfoByHash(entryHash *notaryapi.Hash) (entryInfo *notaryapi.EntryInfo, err error)

	// FetchEntryInfoBranchByHash gets an EntryInfoBranch obj
	FetchEntryInfoBranchByHash(entryHash *notaryapi.Hash) (entryInfoBranch *notaryapi.EntryInfoBranch, err error)

	// FetchEntryBlock gets an entry by hash from the database.
	FetchEBlockByHash(eBlockHash *notaryapi.Hash) (eBlock *notaryapi.EBlock, err error)

	// FetchEBlockByMR gets an entry block by merkle root from the database.
	FetchEBlockByMR(eBMR *notaryapi.Hash) (eBlock *notaryapi.EBlock, err error)

	// FetchEBHashByMR gets an entry by hash from the database.
	FetchEBHashByMR(eBMR *notaryapi.Hash) (eBlockHash *notaryapi.Hash, err error)

	// FetchAllEBlocksByChain gets all of the blocks by chain id
	FetchAllEBlocksByChain(chainID *notaryapi.Hash) (eBlocks *[]notaryapi.EBlock, err error)

	// FetchAllEBInfosByChain gets all of the entry block infos by chain id
	FetchAllEBInfosByChain(chainID *notaryapi.Hash) (eBInfos *[]notaryapi.EBInfo, err error)

	// FetchDBlock gets an entry by hash from the database.
	FetchDBlockByHash(dBlockHash *notaryapi.Hash) (dBlock *notaryapi.DBlock, err error)

	// FetchAllFBInfo gets all of the fbInfo
	FetchAllDBlocks() (fBlocks []notaryapi.DBlock, err error)

	// FetchDBInfoByHash gets all db records related to a directory block
	FetchAllDBRecordsByDBHash(dbHash *notaryapi.Hash, cChainID *notaryapi.Hash) (ldbMap map[string]string, err error)

	// InsertAllDBRecords inserts all key value pairs from map into db
	InsertAllDBRecords(ldbMap map[string]string) (err error)

	// FetchSupportDBRecords gets support db records
	FetchSupportDBRecords() (ldbMap map[string]string, err error)

	// ProcessCBlockBatche inserts the CBlock and update all it's cbentries in DB
	ProcessCBlockBatch(block *notaryapi.CBlock) (err error)

	// FetchAllCBlocks gets all of the entry credit blocks
	FetchAllCBlocks() (cBlocks []notaryapi.CBlock, err error)

	// Initialize External ID map for explorer search
	InitializeExternalIDMap() (extIDMap map[string]bool, err error)

	// ProcessFBlockBatche inserts the FBlock
	ProcessFBlockBatch(block *factoid.FBlock) error

	// FetchCntryBlock gets a block by hash from the database.
	FetchFBlockByHash(fBlockHash *notaryapi.Hash) (fBlock *factoid.FBlock, err error)

	// FetchAllFBlocks gets all of the factoid blocks
	FetchAllFBlocks() (fBlocks []factoid.FBlock, err error)
}
