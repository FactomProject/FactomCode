package wire

import (
	"bytes"
	"encoding/hex"
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

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgDirBlockSig) Command() string {
	return CmdDirBlockSig
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgDirBlockSig) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgDirBlockSig.MsgDecode reader is not a " +
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

// MsgEncode is part of the Message interface implementation.
func (msg *MsgDirBlockSig) MsgEncode(w io.Writer, pver uint32) error {
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

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgDirBlockSig) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// Equals shows if both msg is the same
func (msg *MsgDirBlockSig) Equals(m *MsgDirBlockSig) bool {
	return msg.DBHeight == m.DBHeight &&
		msg.DirBlockHash == m.DirBlockHash
}

// String returns str value
func (msg *MsgDirBlockSig) String() string {
	return fmt.Sprintf("DBHeight=%d, SourceID=%s, DirBlockHash=%s", msg.DBHeight, msg.SourceNodeID, hex.EncodeToString(msg.DirBlockHash.Bytes()))
}
