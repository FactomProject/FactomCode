package ldb

import (
	"github.com/FactomProject/FactomCode/database"
	"log"
	"testing"
)

var db database.Db // database

func TestSimpleOperations(t *testing.T) {

	initDB()

	key1 := []byte("Key1")
	value1 := []byte("Value1")
	key2 := []byte("Key2")
	value2 := []byte("Value2")

	err := db.Put(key1, value1)
	err = db.Put(key2, value2)
	db.Print()
	db.Delete(key1)
	log.Println("after delete:")
	db.Print()

	if err != nil {
		t.Errorf("Error:%v", err)
	}
}

func initDB() {
	var ldbpath = "/tmp/client/ldb8"
	//init db
	var err error
	db, err = OpenLevelDB(ldbpath, false)

	if err != nil {
		log.Println("err opening db: %v", err)
	}

	if db == nil {
		log.Println("Creating new db ...")
		db, err = OpenLevelDB(ldbpath, true)

		if err != nil {
			panic(err)
		}
	}

	log.Println("Database started from: " + ldbpath)

}
