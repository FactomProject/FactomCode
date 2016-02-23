// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"fmt"
	fct "github.com/FactomProject/factoid"
	"io"
)

var _ = fmt.Printf

type IMsgFactoidTX interface {
	// Set the Transaction to be carried by this message.
	SetTransaction(fct.ITransaction)
	// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
	// This is part of the Message interface implementation.
	BtcEncode(w io.Writer, pver uint32) error
	// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
	// This is part of the Message interface implementation.
	BtcDecode(r io.Reader, pver uint32) error
	// Command returns the protocol command string for the message.  This is part
	// of the Message interface implementation.
	Command() string
	// MaxPayloadLength returns the maximum length the payload can be for the
	// receiver.  This is part of the Message interface implementation.
	MaxPayloadLength(pver uint32) uint32
	// NewMsgCommitEntry returns a new bitcoin Commit Entry message that conforms to
	// the Message interface.
	NewMsgFactoidTX() IMsgFactoidTX
	// Check whether the msg can pass the message level validations
	// such as timestamp, signiture and etc
	IsValid() bool
	// Create a sha hash from the message binary (output of BtcEncode)
	Sha() (ShaHash, error)
}

// MsgCommitEntry implements the Message interface and represents a factom
// Commit-Entry message.  It is used by client to commit the entry before
// revealing it.
type MsgFactoidTX struct {
	IMsgFactoidTX
	Transaction fct.ITransaction
}

// Accessor to set the Transaction for a message.
func (msg *MsgFactoidTX) SetTransaction(transaction fct.ITransaction) {
	msg.Transaction = transaction
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgFactoidTX) BtcEncode(w io.Writer, pver uint32) error {

	data, err := msg.Transaction.MarshalBinary()
	if err != nil {
		return err
	}

	err = writeVarBytes(w, pver, data)
	if err != nil {
		return err
	}

	return nil
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgFactoidTX) BtcDecode(r io.Reader, pver uint32) error {

	data, err := readVarBytes(r, pver, uint32(MaxAppMsgPayload), CmdEBlock)
	if err != nil {
		return err
	}

	msg.Transaction = new(fct.Transaction)
	err = msg.Transaction.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgFactoidTX) Command() string {
	return CmdFactoidTX
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgFactoidTX) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewMsgCommitEntry returns a new bitcoin Commit Entry message that conforms to
// the Message interface.
func NewMsgFactoidTX() IMsgFactoidTX {
	return &MsgFactoidTX{}
}

// Check whether the msg can pass the message level validations
// such as timestamp, signiture and etc
func (msg *MsgFactoidTX) IsValid() bool {
	err := msg.Transaction.Validate(1)
	if err != nil {
		return false
	}
	err = msg.Transaction.ValidateSignatures()
	if err != nil {
		return false
	}
	return true
}

// Create a sha hash from the message binary (output of BtcEncode)
func (msg *MsgFactoidTX) Sha() (ShaHash, error) {

	buf := bytes.NewBuffer(nil)
	msg.BtcEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))

	return sha, nil
}
