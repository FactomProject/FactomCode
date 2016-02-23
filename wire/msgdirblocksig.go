package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgDirBlockSig is the msg of the dir block sig after a new dir block is created
type MsgDirBlockSig struct {
	DBHeight     uint32
	DirBlockHash common.Hash
	Sig          common.Signature
	SourceNodeID string
}

// End-of-Minute internal message for time commnunications between Goroutines
func (msg *MsgDirBlockSig) Command() string {
	return CmdDirBlockSig
}

func (msg *MsgDirBlockSig) BtcDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgDirBlockSig.BtcDecode reader is not a " +
			"*bytes.Buffer")
	}
	err := readElements(buf, &msg.DBHeight, &msg.DirBlockHash, &msg.Sig)
	if err != nil {
		return err
	}
	if buf.Len() > 0 {
		nodeID, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.SourceNodeID = nodeID
	}
	return nil
}

func (msg *MsgDirBlockSig) BtcEncode(w io.Writer, pver uint32) error {
	err := writeElements(w, msg.DBHeight, msg.DirBlockHash, msg.Sig)
	if err != nil {
		fmt.Errorf(err.Error())
		return err
	}
	err = writeVarString(w, pver, msg.SourceNodeID)
	if err != nil {
		return err
	}
	return nil
}

func (msg *MsgDirBlockSig) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}
