package consensus


import (
	"testing"
	"github.com/FactomProject/btcd/wire"	

)

func TestPlist(t *testing.T) {

	pl := NewProcessList(10, 0)
	
	ackmsg := wire.NewMsgAcknowledgement(0, 12)
	err := pl.AddToProcessList(ackmsg)

	if err != nil {
		t.Errorf("Error:", err)
	}
}

