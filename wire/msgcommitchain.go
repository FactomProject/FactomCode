// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgCommitEntry implements the Message interface and represents a factom
// Commit-Entry message.  It is used by client to commit the entry before revealing it.
type MsgCommitChain struct {
	CommitChain *common.CommitChain
}

// NewMsgCommitChain returns a new Commit Chain message that conforms to the
// Message interface.  See MsgInv for details.
func NewMsgCommitChain() *MsgCommitChain {
	m := new(MsgCommitChain)
	m.CommitChain = common.NewCommitChain()
	return m
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgCommitChain) BtcEncode(w io.Writer, pver uint32) error {
	bytes, err := msg.CommitChain.MarshalBinary()
	if err != nil {
		return err
	}

	if err := writeVarBytes(w, pver, bytes); err != nil {
		return err
	}

	return nil
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgCommitChain) BtcDecode(r io.Reader, pver uint32) error {
	bytes, err := readVarBytes(r, pver, uint32(common.CommitChainSize),
		CmdEntry)
	if err != nil {
		return err
	}

	msg.CommitChain = common.NewCommitChain()
	if err := msg.CommitChain.UnmarshalBinary(bytes); err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgCommitChain) Command() string {
	return CmdCommitChain
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgCommitChain) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// Check whether the msg can pass the message level validations
// such as timestamp, signiture and etc
func (msg *MsgCommitChain) IsValid() bool {
	return msg.CommitChain.IsValid()
}

// Create a sha hash from the message binary (output of BtcEncode)
func (msg *MsgCommitChain) Sha() (ShaHash, error) {

	buf := bytes.NewBuffer(nil)
	msg.BtcEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))

	return sha, nil
}
