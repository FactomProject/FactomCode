// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgECBlock implements the Message interface and represents a factom ECBlock
// message.  It is used by client to download ECBlock.
type MsgECBlock struct {
	ECBlock *common.ECBlock
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgECBlock) BtcEncode(w io.Writer, pver uint32) error {
	bytes, err := msg.ECBlock.MarshalBinary()
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
func (msg *MsgECBlock) BtcDecode(r io.Reader, pver uint32) error {

	bytes, err := readVarBytes(r, pver, uint32(MaxBlockMsgPayload), CmdECBlock)
	if err != nil {
		return err
	}

	msg.ECBlock = common.NewECBlock()
	err = msg.ECBlock.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgECBlock) Command() string {
	return CmdECBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgECBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockMsgPayload
}

// NewMsgECBlock returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgECBlock() *MsgECBlock {
	return &MsgECBlock{}
}
