package db

import (
	"fmt"

	"github.com/FactomProject/FactomCode/factoid/util"
)

var Db util.Database = NewMemDatabase()

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db map[string][]byte
}

func NewMemDatabase() *MemDatabase {
	db := &MemDatabase{db: make(map[string][]byte)}

	return db
}

func (db *MemDatabase) Put(key []byte, value []byte) {
	db.db[string(key)] = value
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	return db.db[string(key)], nil
}

/*
func (db *MemDatabase) GetKeys() []*util.Key {
	data, _ := db.Get([]byte("KeyRing"))

	return []*util.Key{util.NewKeyFromBytes(data)}
}
*/

func (db *MemDatabase) Delete(key []byte) error {
	delete(db.db, string(key))

	return nil
}

func (db *MemDatabase) Print() {
	for key, val := range db.db {
		fmt.Printf("%x(%d): ", key, len(key))
		node := util.NewValueFromBytes(val)
		fmt.Printf("%q\n", node.Interface())
	}
}

func (db *MemDatabase) Close() {
}

func (db *MemDatabase) LastKnownTD() []byte {
	data, _ := db.Get([]byte("LastKnownTotalDifficulty"))

	if len(data) == 0 || data == nil {
		data = []byte{0x0}
	}

	return data
}
