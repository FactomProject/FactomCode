package consensus

import (
	"github.com/FactomProject/btcd/wire"
	"testing"
)

func TestPlist(t *testing.T) {

	plMgr := NewProcessListMgr(0, 1, 1)

	ackmsg := wire.NewMsgAcknowledgement(0, 12, nil)

	plItem := new(ProcessListItem)
	plItem.ack = ackmsg
	err := plMgr.AddToProcessList(plItem)

	if err != nil {
		t.Errorf("Error:", err)
	}
}
