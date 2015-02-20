// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"github.com/FactomProject/FactomCode/notaryapi"
	"io"
)

// MsgRevealChain implements the Message interface and represents a factom
// Reveal-Chain message.  It is used by client to reveal the chain.
type MsgRevealChain struct {
	Chain *notaryapi.EChain
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgRevealChain) BtcEncode(w io.Writer, pver uint32) error {

	//Chain
	bytes, err := msg.Chain.MarshalBinary()
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
func (msg *MsgRevealChain) BtcDecode(r io.Reader, pver uint32) error {
	//Chain
	bytes, err := readVarBytes(r, pver, MaxAppMsgPayload, CmdRevealChain)
	if err != nil {
		return err
	}

	msg.Chain = new(notaryapi.EChain)
	err = msg.Chain.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgRevealChain) Command() string {
	return CmdRevealChain
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgRevealChain) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewMsgInv returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgRevealChain() *MsgRevealChain {
	return &MsgRevealChain{}
}
