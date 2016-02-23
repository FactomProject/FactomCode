// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgEBlock implements the Message interface and represents a factom
// EBlock message.  It is used by client to download the EBlock.
type MsgEBlock struct {
	EBlk *common.EBlock
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgEBlock) BtcEncode(w io.Writer, pver uint32) error {

	bytes, err := msg.EBlk.MarshalBinary()
	if err != nil {
		return err
	}

	err = writeVarBytes(w, pver, bytes)
	if err != nil {
		return err
	}

	return nil
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgEBlock) BtcDecode(r io.Reader, pver uint32) error {

	bytes, err := readVarBytes(r, pver, uint32(MaxBlockMsgPayload), CmdEBlock)
	if err != nil {
		return err
	}

	msg.EBlk = common.NewEBlock()
	err = msg.EBlk.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgEBlock) Command() string {
	return CmdEBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgEBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockMsgPayload
}

// NewMsgEBlock returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgEBlock() *MsgEBlock {
	return &MsgEBlock{}
}
