package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgCurrentLeader is used to select the current leader replacement
// in case of it's gone
type MsgCurrentLeader struct {
	CurrLeaderGone      string // it's gone but still needs to be verified
	NewLeaderCandidates string
	SourceNodeID        string
	StartDBHeight       uint32
	Sig                 common.Signature
}

// Command is CmdCurrentLeader
func (msg *MsgCurrentLeader) Command() string {
	return CmdCurrentLeader
}

// BtcDecode is part of the Message interface implementation.
func (msg *MsgCurrentLeader) BtcDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgCurrentLeader.BtcDecode reader is not a " +
			"*bytes.Buffer")
	}
	if buf.Len() > 0 {
		cid, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.CurrLeaderGone = cid
	}
	if buf.Len() > 0 {
		nid, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.NewLeaderCandidates = nid
	}
	if buf.Len() > 0 {
		cid, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.SourceNodeID = cid
	}
	err := readElements(buf, &msg.StartDBHeight, &msg.Sig)
	if err != nil {
		return err
	}
	return nil
}

// BtcEncode is part of the Message interface implementation.
func (msg *MsgCurrentLeader) BtcEncode(w io.Writer, pver uint32) error {
	err := writeVarString(w, pver, msg.CurrLeaderGone)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.NewLeaderCandidates)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.SourceNodeID)
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
func (msg *MsgCurrentLeader) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewCurrentLeaderMsg creates a new MsgCurrentLeader
func NewCurrentLeaderMsg(currLeaderID string, candidates string, sourceID string,
	height uint32, sig common.Signature) *MsgCurrentLeader {
	return &MsgCurrentLeader{
		CurrLeaderGone:      currLeaderID,
		NewLeaderCandidates: candidates,
		SourceNodeID:        sourceID,
		StartDBHeight:       height,
		Sig:                 sig,
	}
}

// String returns its string value
func (msg *MsgCurrentLeader) String() string {
	return fmt.Sprintf("MsgCurrentLeader: CurrLeaderID=%s, candidates=%s, sourceID=%s, StartDBHeight=%d",
		msg.CurrLeaderGone, msg.NewLeaderCandidates, msg.SourceNodeID, msg.StartDBHeight)
}
