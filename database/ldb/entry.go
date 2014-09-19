// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ldb

import (
//	"bytes"
//	"encoding/binary"

	"github.com/FactomProject/FactomCode/notaryapi"
)


// InsertEntry inserts an entry hash and its associated data into the database.
func (db *LevelDb) InsertEntry(entrySha *notaryapi.Hash, binaryEntry []byte) (err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	
	return db.lDb.Put(entrySha.Bytes, binaryEntry, db.wo)
} 

// FetchEntry gets an entry by hash from the database.
func (db *LevelDb) FetchEntryByHash(entrySha *notaryapi.Hash) (entry notaryapi.Entry, err error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()
	data, err := db.lDb.Get(entrySha.Bytes, db.ro)
	
//	entry = * new (notaryapi.Entry)
	entry.UnmarshalBinary(data)
	
	return entry, nil
} 

