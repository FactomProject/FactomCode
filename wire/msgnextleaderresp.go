package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgNextLeaderResp is sent when a leaderElected confirms its own candidacy
type MsgNextLeaderResp struct {
	CurrLeaderID  string
	NextLeaderID  string
	SourceNodeID  string
	StartDBHeight uint32
	Sig           common.Signature
	Confirmed     bool
}

// Command is CmdNextLeaderResp
func (msg *MsgNextLeaderResp) Command() string {
	return CmdNextLeaderResp
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgNextLeaderResp) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgNextLeaderResp.MsgDecode reader is not a " +
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
	if buf.Len() > 0 {
		sid, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.SourceNodeID = sid
	}
	err := readElements(buf, &msg.StartDBHeight, &msg.Sig, &msg.Confirmed)
	if err != nil {
		return err
	}
	return nil
}

// MsgEncode is part of the Message interface implementation.
func (msg *MsgNextLeaderResp) MsgEncode(w io.Writer, pver uint32) error {
	err := writeVarString(w, pver, msg.CurrLeaderID)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.NextLeaderID)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.SourceNodeID)
	if err != nil {
		return err
	}
	err = writeElements(w, msg.StartDBHeight, msg.Sig, msg.Confirmed)
	if err != nil {
		fmt.Errorf(err.Error())
		return err
	}
	return nil
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgNextLeaderResp) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewNextLeaderRespMsg creates a new MsgNextLeaderResp
func NewNextLeaderRespMsg(currLeaderID string, nextLeaderID string, sid string,
	height uint32, sig common.Signature, confirmed bool) *MsgNextLeaderResp {
	return &MsgNextLeaderResp{
		CurrLeaderID:  currLeaderID,
		NextLeaderID:  nextLeaderID,
		SourceNodeID:  sid,
		StartDBHeight: height,
		Sig:           sig,
		Confirmed:     confirmed,
	}
}

// String returns its string value
func (msg *MsgNextLeaderResp) String() string {
	return fmt.Sprintf("MsgNextLeaderResp: Curr=%s, Next=%s, Start=%d",
		msg.CurrLeaderID, msg.NextLeaderID, msg.StartDBHeight)
}

// Sha Creates a sha hash from the message binary (output of MsgEncode)
func (msg *MsgNextLeaderResp) Sha() (ShaHash, error) {
	buf := bytes.NewBuffer(nil)
	msg.MsgEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))
	return sha, nil
}
