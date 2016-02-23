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
// Commit-Entry message.  It is used by client to commit the entry before
// revealing it.
type MsgCommitEntry struct {
	CommitEntry *common.CommitEntry
}

// NewMsgCommitEntry returns a new Commit Entry message that conforms to the
// Message interface.
func NewMsgCommitEntry() *MsgCommitEntry {
	m := new(MsgCommitEntry)
	m.CommitEntry = common.NewCommitEntry()
	return m
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgCommitEntry) BtcEncode(w io.Writer, pver uint32) error {
	bytes, err := msg.CommitEntry.MarshalBinary()
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
func (msg *MsgCommitEntry) BtcDecode(r io.Reader, pver uint32) error {
	bytes, err := readVarBytes(r, pver, uint32(common.CommitEntrySize),
		CmdEntry)
	if err != nil {
		return err
	}

	msg.CommitEntry = common.NewCommitEntry()
	err = msg.CommitEntry.UnmarshalBinary(bytes)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgCommitEntry) Command() string {
	return CmdCommitEntry
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgCommitEntry) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// Check whether the msg can pass the message level validations
// such as timestamp, signiture and etc
func (msg *MsgCommitEntry) IsValid() bool {
	return msg.CommitEntry.IsValid()
}

// Create a sha hash from the message binary (output of BtcEncode)
func (msg *MsgCommitEntry) Sha() (ShaHash, error) {

	buf := bytes.NewBuffer(nil)
	msg.BtcEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))

	return sha, nil
}
