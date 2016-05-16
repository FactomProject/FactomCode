package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgCandidate shows the dbheight when a candidate is turned into a follower officically
type MsgCandidate struct {
	DBHeight     uint32		// DBHeight that this node turns from Candidate to Follower
	Sig          common.Signature
	SourceNodeID string
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgCandidate) Command() string {
	return CmdCandidate
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgCandidate) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgCandidate.MsgDecode reader is not a " +
			"*bytes.Buffer")
	}
	err := readElements(buf, &msg.DBHeight, &msg.Sig)
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
func (msg *MsgCandidate) MsgEncode(w io.Writer, pver uint32) error {
	err := writeElements(w, msg.DBHeight, msg.Sig)
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
func (msg *MsgCandidate) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// Equals shows if both msg is the same
func (msg *MsgCandidate) Equals(m *MsgCandidate) bool {
	return msg.DBHeight == m.DBHeight &&
		msg.SourceNodeID == m.SourceNodeID
}

// NewMsgCandidate returns a new ack message that conforms to the Message
// interface.
func NewMsgCandidate(height uint32, id string, sig common.Signature) *MsgCandidate {
	return &MsgCandidate{
		DBHeight:     height,
		SourceNodeID: id,
		Sig:          sig,
	}
}

// String returns str value
func (msg *MsgCandidate) String() string {
	return fmt.Sprintf("MsgCandidate( h=%d, from=%s)", msg.DBHeight, msg.SourceNodeID)
}
