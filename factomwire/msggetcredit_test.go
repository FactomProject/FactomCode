package factomwire

/*
import (
	"encoding/hex"
	"github.com/FactomProject/FactomCode/notaryapi"
	"testing"
	"bytes"
	"fmt"
//	"io"
)

func TestGetCredit(t *testing.T) {
	hexkey := "ed14447c656241bf7727fce2e2a48108374bec6e71358f0a280608b292c7f3bc"
	binkey, _ := hex.DecodeString(hexkey)
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(binkey)

	barray1 := (make([]byte, 32))
	barray1[31] = 2
	factoidTxHash := new (notaryapi.Hash)
	factoidTxHash.SetBytes(barray1)

	//Write msg
	msgOutgoing := NewMsgGetCredit()
	msgOutgoing.ECPubKey = pubKey
	msgOutgoing.FactoidBase = uint64(20000)
	fmt.Printf("msgOutgoing:%+v\n", msgOutgoing)

    var buf bytes.Buffer
	msgOutgoing.BtcEncode(&buf, uint32(1))
	fmt.Println("Outgoing msg bytes: ", buf.Bytes())

	//Read msg
	msgIncoming:= NewMsgGetCredit()
	err:=msgIncoming.BtcDecode(&buf, uint32(1))

	fmt.Printf("msgIncoming:%+v\n", msgIncoming)

	if err != nil {
		t.Errorf("Error:", err)
	}

}
*/
