package restapi

import (
	"fmt"
	"github.com/FactomProject/FactomCode/database/ldb"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
	"log"
	"testing"
	//	"time"
	"encoding/hex"
)

func TestBuyCredit(t *testing.T) {

	initDB()
	inMsgQ := make(chan factomwire.Message, 100)  //incoming message queue for factom application messages
	outMsgQ := make(chan factomwire.Message, 100) //outgoing message queue for factom application messages

	hexkey := "ed14447c656241bf7727fce2e2a48108374bec6e71358f0a280608b292c7f3bc"
	binkey, _ := hex.DecodeString(hexkey)
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(binkey)

	//Write msg
	msgOutgoing := factomwire.NewMsgGetCredit()
	msgOutgoing.ECPubKey = pubKey
	msgOutgoing.FactoidBase = uint64(2000000000)
	fmt.Printf("msgOutgoing:%+v\n", msgOutgoing)

	inMsgQ <- msgOutgoing

	Start_Processor(db, inMsgQ, outMsgQ)
}

/*
func TestAddChain(t *testing.T) {

	chain := new (notaryapi.EChain)
	bName := make ([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping2"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new (notaryapi.Entry)
	entry.ChainID = *chain.ChainID
	entry.ExtIDs = make ([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))
	//entry.ExtIDs = append(entry.Extcd IDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))
	//entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("First entry for chain:\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\"Rules:\"asl;djfasldkfjasldfjlksouiewopurw111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111\"")

	chain.FirstEntry = entry

	binaryEntry, _ := entry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)

	entryChainIDHash := notaryapi.Sha(append(chain.ChainID.Bytes, entryHash.Bytes ...))

	// Calculate the required credits
	binaryChain, _ := chain.MarshalBinary()
	credits := int32(binary.Size(binaryChain)/1000 + 1) + creditsPerChain

	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)
	printCreditMap()
	printPaidEntryMap()
	printCChain()
	_, err := processCommitChain(entryHash, chain.ChainID, entryChainIDHash, pubKey, credits)
	fmt.Println("after processCommitChain:")
	printPaidEntryMap()

	if err != nil {
		fmt.Println("Error:", err)
	}

	// Reveal new chain
	_, err = processRevealChain(chain)
	fmt.Println("after processNewChain:")
	printPaidEntryMap()
	if err != nil {
		fmt.Println("Error:", err)
	}

}

func TestAddEntry(t *testing.T) {
	chain := new (notaryapi.EChain)
	bName := make ([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping2"))

	chain.Name = bName
	chain.GenerateIDFromName()



	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)

	for i:=1; i<400000; i++{

		entry := new (notaryapi.Entry)
		entry.ExtIDs = make ([][]byte, 0, 5)
		entry.ExtIDs = append(entry.ExtIDs, []byte(string(i)))
		entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))
		entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
		entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222\"")
		entry.ChainID = *chain.ChainID

		binaryEntry, _ := entry.MarshalBinary()
		entryHash := notaryapi.Sha(binaryEntry)
		timestamp := int64(i)
		// Calculate the required credits
		credits := int32(binary.Size(binaryEntry)/1000 + 1)

		_, err := processCommitEntry(entryHash, pubKey, timestamp, credits)
		fmt.Println("after processCommitEntry:")
		printCreditMap()
		printPaidEntryMap()
//		printCChain()
		if err != nil {
			t.Errorf("Error:", err)
		}

		// Reveal new entry
		processRevealEntry(entry)
		fmt.Println("after processRevealEntry:")
		printPaidEntryMap()
		if err != nil {
			t.Errorf("Error:", err)
		}
		time.Sleep(time.Microsecond * 1)

	}

}
*/
func initDB() {

	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)

	if err != nil {
		log.Println("err opening db: %v", err)

	}

	if db == nil {
		log.Println("Creating new db ...")
		db, err = ldb.OpenLevelDB(ldbpath, true)

		if err != nil {
			panic(err)
		}
	}
	log.Println("Database started from: " + ldbpath)

}
