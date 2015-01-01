
package ldb

import (
	"fmt"
)


func (db *LevelDb) Put(key []byte, value []byte) error{
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	return db.lDb.Put(key, value, nil)
	
}

func (db *LevelDb) Get(key []byte) ([]byte, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()	
		
	return db.lDb.Get(key, db.ro)
}

func (db *LevelDb) Delete(key []byte) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()	
	
	return db.lDb.Delete(key, db.wo)
}

func (db *LevelDb) LastKnownTD() []byte {
	data, _ := db.lDb.Get([]byte("LTD"), nil)

	if len(data) == 0 {
		data = []byte{0x0}
	}

	return data
}

func (db *LevelDb) Print() {
	iter := db.lDb.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		fmt.Printf("%x(%d): ", key, len(key))
		fmt.Println("%x(%d) ", value, len(value))		
	}
}