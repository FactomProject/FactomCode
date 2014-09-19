package database

import (

	"github.com/FactomProject/FactomCode/notaryapi"	
)
 

// AllShas is a special value that can be used as the final sha when requesting
// a range of shas by height to request them all.
const AllShas = int64(^uint64(0) >> 1)

// Db defines a generic interface that is used to request and insert data into
// the bitcoin block chain.  This interface is intended to be agnostic to actual
// mechanism used for backend data storage.  The AddDBDriver function can be
// used to add a new backend data storage method.
type Db interface {
	// Close cleanly shuts down the database and syncs all data.
	Close() (err error)
	

	// InsertEntry inserts an entry hash and its associated data into the database.
	InsertEntry(entrySha *notaryapi.Hash, binaryEntry []byte) (err error)
	
	// FetchEntry gets an entry by hash from the database.
	FetchEntryByHash(entrySha *notaryapi.Hash) (entry notaryapi.Entry, err error)



	// RollbackClose discards the recent database changes to the previously
	// saved data at last Sync and closes the database.
	RollbackClose() (err error)

	// Sync verifies that the database is coherent on disk and no
	// outstanding transactions are in flight.
	Sync() (err error)
}

