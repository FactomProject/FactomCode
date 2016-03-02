package wire_test

import (
	"bytes"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/wire"
	"github.com/davecgh/go-spew/spew"
	"net"
	"testing"
	"time"
)

func TestMsgVersion(t *testing.T) {
	t.Log("\nTestMsgVersion===========================================================================")

	// MsgVersion.
	addrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you, err := wire.NewNetAddress(addrYou, wire.SFNodeNetwork)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}
	you.Timestamp = time.Time{} // Version message has zero value timestamp.
	addrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me, err := wire.NewNetAddress(addrMe, wire.SFNodeNetwork)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}
	me.Timestamp = time.Time{} // Version message has zero value timestamp.
	msgVersion := wire.NewMsgVersion(me, you, 123123, 0)

	// Sign the ack using server private keys
	serverPrivKey, err := common.NewPrivateKeyFromHex("07c0d52cb74f4ca3106d80c4a70488426886bccc6ebc10c6bafb37bf8a65f4c38cee85c62a9e48039d4ac294da97943c2001be1539809ea5f54721f0c5477a0a")
	msgVersion.NodeSig = serverPrivKey.Sign([]byte(msgVersion.NodeID))

	//t.Errorf("ReadMessage =, msg %v", 		spew.Sdump(msgVersion))

	buf := bytes.Buffer{}
	err = msgVersion.BtcEncode(&buf, 1)
	if err != nil {
		t.Errorf("Error:", err)
	}

	b1 := buf.Bytes()
	err = msgVersion.BtcDecode(&buf, 1)

	buf2 := bytes.Buffer{}
	err = msgVersion.BtcEncode(&buf2, 1)
	if err != nil {
		t.Errorf("Error:", err)
	}

	b2 := buf2.Bytes()

	//t.Errorf("ReadMessage =, msg %v", 		spew.Sdump(b2))

	if bytes.Compare(b1, b2) != 0 {
		t.Errorf("Invalid output")
	}

}
