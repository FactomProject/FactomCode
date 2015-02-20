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

func TestCommitChain(t *testing.T) {
	fmt.Println("\nTestCommitChain===========================================================================")
	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping2"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new(notaryapi.Entry)
	entry.ChainID = *chain.ChainID
	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))
	entry.Data = []byte("First entry for chain:\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\"Rules:\"asl;djfasldkfjasldfjlksouiewopurw111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111\"")

	chain.FirstEntry = entry

	binaryEntry, _ := entry.MarshalBinary()
	entryHash := notaryapi.Sha(binaryEntry)

	entryChainIDHash := notaryapi.Sha(append(chain.ChainID.Bytes, entryHash.Bytes...))

	// Calculate the required credits
	binaryChain, _ := chain.MarshalBinary()
	credits := uint32(binary.Size(binaryChain)/1000+1) + 10

	timestamp := uint64(time.Now().Unix())
	var msg bytes.Buffer
	binary.Write(&msg, binary.BigEndian, timestamp)
	msg.Write(chain.ChainID.Bytes)
	msg.Write(entryHash.Bytes)
	msg.Write(entryChainIDHash.Bytes)

	binary.Write(&msg, binary.BigEndian, credits)

	sig := wallet.SignData(msg.Bytes())

	hexkey := "ed14447c656241bf7727fce2e2a48108374bec6e71358f0a280608b292c7f3bc"
	binkey, _ := hex.DecodeString(hexkey)
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(binkey)

	//Write msg
	msgOutgoing := factomwire.NewMsgCommitChain()
	msgOutgoing.ECPubKey = pubKey
	msgOutgoing.ChainID = chain.ChainID
	msgOutgoing.EntryHash = entryHash
	msgOutgoing.EntryChainIDHash = entryChainIDHash
	msgOutgoing.Credits = credits
	msgOutgoing.Timestamp = timestamp
	msgOutgoing.Sig = sig.Sig[:]
	fmt.Printf("msgOutgoing:%+v\n", msgOutgoing)

	var buf bytes.Buffer
	msgOutgoing.BtcEncode(&buf, uint32(1))
	fmt.Println("Outgoing msg bytes: ", buf.Bytes())

	//Read msg
	msgIncoming := factomwire.NewMsgCommitChain()
	err := msgIncoming.BtcDecode(&buf, uint32(1))

	fmt.Printf("msgIncoming:%+v\n", msgIncoming)

	if err != nil {
		t.Errorf("Error:", err)
	}

}
