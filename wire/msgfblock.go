// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"github.com/FactomProject/factoid/block"
	"io"
)

// factoid block
type MsgFBlock struct {
	SC block.IFBlock
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgFBlock) BtcEncode(w io.Writer, pver uint32) error {

	bytes, err := msg.SC.MarshalBinary()
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
func (msg *MsgFBlock) BtcDecode(r io.Reader, pver uint32) error {

	bytes, err := readVarBytes(r, pver, uint32(MaxBlockMsgPayload), CmdFBlock)
	if err != nil {
		return err
	}

	msg.SC = new(block.FBlock)
	err = msg.SC.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgFBlock) Command() string {
	return CmdFBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgFBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockMsgPayload
}

// NewMsgABlock returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgFBlock() *MsgFBlock {
	return &MsgFBlock{}
}
