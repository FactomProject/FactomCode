package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgCurrentLeaderResp is the response to select the current leader replacement
// in case of it's gone
type MsgCurrentLeaderResp struct {
	CurrLeaderGone      string // it's gone but still needs to be verified
	NewLeaderCandidate	string
	SourceNodeID        string
	StartDBHeight       uint32
	Sig                 common.Signature
}

// Command is CmdCurrentLeaderResp
func (msg *MsgCurrentLeaderResp) Command() string {
	return CmdCurrentLeaderResp
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgCurrentLeaderResp) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgCurrentLeaderResp.MsgDecode reader is not a " +
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
		msg.NewLeaderCandidate = nid
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

// MsgEncode is part of the Message interface implementation.
func (msg *MsgCurrentLeaderResp) MsgEncode(w io.Writer, pver uint32) error {
	err := writeVarString(w, pver, msg.CurrLeaderGone)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.NewLeaderCandidate)
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
func (msg *MsgCurrentLeaderResp) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewCurrentLeaderRespMsg creates a new MsgCurrentLeaderResp
func NewCurrentLeaderRespMsg(currLeaderID string, candidates string, sourceID string,
	height uint32, sig common.Signature) *MsgCurrentLeaderResp {
	return &MsgCurrentLeaderResp{
		CurrLeaderGone:      currLeaderID,
		NewLeaderCandidate: candidates,
		SourceNodeID:        sourceID,
		StartDBHeight:       height,
		Sig:                 sig,
	}
}

// String returns its string value
func (msg *MsgCurrentLeaderResp) String() string {
	return fmt.Sprintf("MsgCurrentLeaderResp: CurrLeaderID=%s, candidates=%s, sourceID=%s, StartDBHeight=%d",
		msg.CurrLeaderGone, msg.NewLeaderCandidate, msg.SourceNodeID, msg.StartDBHeight)
}
