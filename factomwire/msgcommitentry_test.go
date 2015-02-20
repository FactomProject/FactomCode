package factomwire_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/wallet"
	"testing"
	"time"
)

func TestCommitEntry(t *testing.T) {
	fmt.Println("\nTestCommitEntry===========================================================================")
	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping2"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new(notaryapi.Entry)
	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte(string(1)))
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222\"")
	entry.ChainID = *chain.ChainID

	binaryEntry, _ := entry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)

	// Calculate the required credits
	credits := uint32(binary.Size(binaryEntry)/1000 + 1)

	timestamp := uint64(time.Now().Unix())
	var msg bytes.Buffer
	binary.Write(&msg, binary.BigEndian, timestamp)
	msg.Write(entryHash.Bytes)
	binary.Write(&msg, binary.BigEndian, credits)

	sig := wallet.SignData(msg.Bytes())

	hexkey := "ed14447c656241bf7727fce2e2a48108374bec6e71358f0a280608b292c7f3bc"
	binkey, _ := hex.DecodeString(hexkey)
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(binkey)

	//Write msg
	msgOutgoing := factomwire.NewMsgCommitEntry()
	msgOutgoing.ECPubKey = pubKey
	msgOutgoing.EntryHash = entryHash
	msgOutgoing.Credits = credits
	msgOutgoing.Timestamp = timestamp
	msgOutgoing.Sig = sig.Sig[:]
	fmt.Printf("msgOutgoing:%+v\n", msgOutgoing)

	var buf bytes.Buffer
	msgOutgoing.BtcEncode(&buf, uint32(1))
	fmt.Println("Outgoing msg bytes: ", buf.Bytes())

	//Read msg
	msgIncoming := factomwire.NewMsgCommitEntry()
	err := msgIncoming.BtcDecode(&buf, uint32(1))

	fmt.Printf("msgIncoming:%+v\n", msgIncoming)

	if err != nil {
		t.Errorf("Error:", err)
	}

}
