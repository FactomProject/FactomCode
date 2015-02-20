package restapi

/*
import (
	"encoding/hex"
	"github.com/FactomProject/FactomCode/notaryapi"
	"testing"
)

func TestBuyCredit(t *testing.T) {
	hexkey := "ed14447c656241bf7727fce2e2a48108374bec6e71358f0a280608b292c7f3bc"
	binkey, _ := hex.DecodeString(hexkey)
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(binkey)

	barray1 := (make([]byte, 32))
	barray1[31] = 2
	factoidTxHash := new (notaryapi.Hash)
	factoidTxHash.SetBytes(barray1)

	_, err := processBuyEntryCredit(pubKey, 200000, factoidTxHash)

	if err != nil {
		t.Errorf("Error:", err)
	}
}
*/
