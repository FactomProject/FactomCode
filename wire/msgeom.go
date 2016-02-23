package wire

import (
	"bytes"
	"fmt"
	"io"
)

// End-of-Minute message for time commnunications between Goroutines
type MsgEOM struct {
	EOMType          byte
	NextDBlockHeight uint32
}

// End-of-Minute internal message for time commnunications between Goroutines
func (msg *MsgEOM) Command() string {
	return CmdEOM
}

func (msg *MsgEOM) BtcDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgVersion.BtcDecode reader is not a " +
			"*bytes.Buffer")
	}
	err := readElements(buf, &msg.EOMType, &msg.NextDBlockHeight)
	if err != nil {
		return err
	}
	return nil
}

func (msg *MsgEOM) BtcEncode(w io.Writer, pver uint32) error {
	err := writeElements(w, msg.EOMType, msg.NextDBlockHeight)
	if err != nil {
		fmt.Errorf(err.Error())
		return err
	}
	return nil
}

func (msg *MsgEOM) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}
