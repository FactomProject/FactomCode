package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgNextLeader is the msg used to select the next leader
type MsgNextLeader struct {
	CurrLeaderID  string
	NextLeaderID  string
	StartDBHeight uint32
	Sig           common.Signature
}

// Command is CmdNextLeader
func (msg *MsgNextLeader) Command() string {
	return CmdNextLeader
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgNextLeader) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgNextLeader.MsgDecode reader is not a " +
			"*bytes.Buffer")
	}
	if buf.Len() > 0 {
		cid, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.CurrLeaderID = cid
	}
	if buf.Len() > 0 {
		nid, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.NextLeaderID = nid
	}
	err := readElements(buf, &msg.StartDBHeight, &msg.Sig)
	if err != nil {
		return err
	}
	return nil
}

// MsgEncode is part of the Message interface implementation.
func (msg *MsgNextLeader) MsgEncode(w io.Writer, pver uint32) error {
	err := writeVarString(w, pver, msg.CurrLeaderID)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.NextLeaderID)
	if err != nil {
		return err
	}
	err = writeElements(w, msg.StartDBHeight, msg.Sig)
	if err != nil {
		fmt.Errorf(err.Error())
		return err
	}
	return nil
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgNextLeader) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewNextLeaderMsg creates a new MsgNextLeader
func NewNextLeaderMsg(currLeaderID string, nextLeaderID string,
	height uint32, sig common.Signature) *MsgNextLeader {
	return &MsgNextLeader{
		CurrLeaderID:  currLeaderID,
		NextLeaderID:  nextLeaderID,
		StartDBHeight: height,
		Sig:           sig,
	}
}

// String returns its string value
func (msg *MsgNextLeader) String() string {
	return fmt.Sprintf("MsgNextLeader: CurrLeaderID=%s, NextLeaderID=%s, StartDBHeight=%d",
		msg.CurrLeaderID, msg.NextLeaderID, msg.StartDBHeight)
}
