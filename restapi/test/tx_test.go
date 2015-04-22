package restapi

import (
	"fmt"
	"testing"
)

var (
	_ = fmt.Sprint("testing")
)

func TestSendRawTransactionToBTC(t *testing.T) {
	if err := initWallet(); err != nil {
		t.Error(err)
	}
	
	msg := []byte("testing string")
	hash, err := SendRawTransactionToBTC(msg)
	if err != nil {
		t.Error(err)
	}
	if hash == nil {
		t.Error("hash is nil")
	}
	fmt.Println(hash)
}