package ldb

import (
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/notaryapi"	
	"log"
	"testing"
)

var db database.Db // database

func TestSimpleOperations(t *testing.T) {

	initDB()

	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping3"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new(notaryapi.Entry)
	entry.ChainID = *chain.ChainID
	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw\"")
	
	entryBinary, _ := entry.MarshalBinary()
	entryHash := notaryapi.Sha(entryBinary)
	err := db.InsertEntryAndQueue(entryHash, &entryBinary, entry, &chain.ChainID.Bytes)	
	
	entry1, _ := db.FetchEntryByHash(entryHash)
	
	t.Logf("entry1: %+v", entry1)
	
	t.Logf("entry1.data: %+v", string(entry1.Data))

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
