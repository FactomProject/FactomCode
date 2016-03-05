package wire_test

import (
	"bytes"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/consensus"
	"github.com/FactomProject/FactomCode/wire"
	"testing"
)

func TestMsgACK(t *testing.T) {
	t.Log("\nTestAck===========================================================================")
	ack := wire.NewMsgAck(1, 2, nil, wire.EndMinute10)

	// Sign the ack using server private keys
	b, _ := ack.GetBinaryForSignature()
	serverPrivKey, err := common.NewPrivateKeyFromHex("07c0d52cb74f4ca3106d80c4a70488426886bccc6ebc10c6bafb37bf8a65f4c38cee85c62a9e48039d4ac294da97943c2001be1539809ea5f54721f0c5477a0a")
	plMgr := consensus.NewProcessListMgr(1, 1, 10, serverPrivKey)
	ack.Signature = *plMgr.SignAck(b).Sig

	buf := bytes.Buffer{}
	err = ack.BtcEncode(&buf, 1)
	if err != nil {
		t.Errorf("Error:", err)
	}

	b1 := buf.Bytes()
	err = ack.BtcDecode(&buf, 1)

	buf2 := bytes.Buffer{}
	err = ack.BtcEncode(&buf2, 1)
	if err != nil {
		t.Errorf("Error:", err)
	}

	b2 := buf2.Bytes()

	if bytes.Compare(b1, b2) != 0 {
		t.Errorf("Invalid output")
	}

}
