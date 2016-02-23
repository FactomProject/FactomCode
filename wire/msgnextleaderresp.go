package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgNextLeaderResp is the msg of the dir block sig after a new dir block is created
type MsgNextLeaderResp struct {
	CurrLeaderID  string
	NextLeaderID  string
	StartDBHeight uint32
	Sig           common.Signature
	Confirmed     bool
}

// Command is CmdNextLeader
func (msg *MsgNextLeaderResp) Command() string {
	return CmdNextLeaderResp
}

func (msg *MsgNextLeaderResp) BtcDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgNextLeaderResp.BtcDecode reader is not a " +
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
	err := readElements(buf, &msg.StartDBHeight, &msg.Sig, &msg.Confirmed)
	if err != nil {
		return err
	}
	return nil
}

func (msg *MsgNextLeaderResp) BtcEncode(w io.Writer, pver uint32) error {
	err := writeVarString(w, pver, msg.CurrLeaderID)
	if err != nil {
		return err
	}
	err = writeVarString(w, pver, msg.NextLeaderID)
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

func (msg *MsgNextLeaderResp) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

func NewNextLeaderRespMsg(currLeaderID string, nextLeaderID string,
	height uint32, sig common.Signature, confirmed bool) *MsgNextLeaderResp {
	return &MsgNextLeaderResp{
		CurrLeaderID:  currLeaderID,
		NextLeaderID:  nextLeaderID,
		StartDBHeight: height,
		Sig:           sig,
		Confirmed:     confirmed,
	}
}
