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
	NewLeaderCandidate	string
	AddrCandidate		NetAddress
	SourceNodeID        string
	StartDBHeight       uint32
	Sig                 common.Signature
}

// Command is CmdCurrentLeader
func (msg *MsgCurrentLeader) Command() string {
	return CmdCurrentLeader
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgCurrentLeader) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgCurrentLeader.MsgDecode reader is not a " +
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
		err := readNetAddress(buf, pver, &msg.AddrCandidate, false)
		if err != nil {
			return err
		}
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
func (msg *MsgCurrentLeader) MsgEncode(w io.Writer, pver uint32) error {
	err := writeVarString(w, pver, msg.CurrLeaderGone)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.NewLeaderCandidate)
	if err != nil {
		return err
	}
	err = writeNetAddress(w, pver, &msg.AddrCandidate, false)
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
		NewLeaderCandidate: candidates,
		SourceNodeID:        sourceID,
		StartDBHeight:       height,
		Sig:                 sig,
	}
}

// String returns its string value
func (msg *MsgCurrentLeader) String() string {
	return fmt.Sprintf("MsgCurrentLeader(curr=%s, new=%s, from=%s, start=%d)",
		msg.CurrLeaderGone, msg.NewLeaderCandidate, msg.SourceNodeID, msg.StartDBHeight)
}

// Sha Creates a sha hash from the message binary (output of MsgEncode)
func (msg *MsgCurrentLeader) Sha() (ShaHash, error) {
	buf := bytes.NewBuffer(nil)
	msg.MsgEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))
	return sha, nil
}
