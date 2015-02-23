// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"github.com/FactomProject/FactomCode/notaryapi"
	"io"
)

type MsgConfirmation struct {
	Height      uint64
	ChainID     notaryapi.Hash
	Index       uint32
	Affirmation [32]byte
	Hash        [32]byte
	Signature   [64]byte
	MsgHash     *ShaHash // hash of the msg being confirmed, unsure how I can use it yet
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgConfirmation) BtcDecode(r io.Reader, pver uint32) error {
	err := readElements(r, &msg.Height, &msg.ChainID, &msg.Index, &msg.Affirmation, &msg.Hash, &msg.Signature, &msg.MsgHash)
	if err != nil {
		return err
	}

	return nil
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgConfirmation) BtcEncode(w io.Writer, pver uint32) error {
	err := writeElements(w, &msg.Height, &msg.ChainID, &msg.Index, &msg.Affirmation, &msg.Hash, &msg.Signature, &msg.MsgHash)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgConfirmation) Command() string {
	return CmdConfirmation
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgConfirmation) MaxPayloadLength(pver uint32) uint32 {

	// 10K is too big of course, TODO: adjust
	return MaxAppMsgPayload
}

// NewMsgConfirmation returns a new bitcoin ping message that conforms to the Message
// interface.  See MsgConfirmation for details.
func NewMsgConfirmation(height uint64, index uint32) *MsgConfirmation {
	return &MsgConfirmation{
		Height: height,
		Index:  index,
	}
}
