// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgABlock implements the Message interface and represents a factom
// Admin Block message.  It is used by client to download Admin Block.
type MsgABlock struct {
	ABlk *common.AdminBlock
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgABlock) BtcEncode(w io.Writer, pver uint32) error {

	bytes, err := msg.ABlk.MarshalBinary()
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
func (msg *MsgABlock) BtcDecode(r io.Reader, pver uint32) error {

	bytes, err := readVarBytes(r, pver, uint32(MaxBlockMsgPayload), CmdABlock)
	if err != nil {
		return err
	}

	msg.ABlk = new(common.AdminBlock)
	err = msg.ABlk.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgABlock) Command() string {
	return CmdABlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgABlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockMsgPayload
}

// NewMsgABlock returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgABlock() *MsgABlock {
	return &MsgABlock{}
}
