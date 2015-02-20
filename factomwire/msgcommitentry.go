// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/agl/ed25519"
	"io"
)

// MsgCommitEntry implements the Message interface and represents a factom
// Commit-Entry message.  It is used by client to commit the entry before revealing it.
type MsgCommitEntry struct {
	ECPubKey  *notaryapi.Hash
	EntryHash *notaryapi.Hash
	Credits   uint32
	Timestamp uint64
	Sig       []byte
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgCommitEntry) BtcEncode(w io.Writer, pver uint32) error {

	//ECPubKey
	err := writeVarBytes(w, uint32(notaryapi.HashSize), msg.ECPubKey.Bytes)
	if err != nil {
		return err
	}

	//EntryHash
	err = writeVarBytes(w, uint32(notaryapi.HashSize), msg.EntryHash.Bytes)
	if err != nil {
		return err
	}

	//Credits
	err = writeElement(w, &msg.Credits) // change it to varint??
	if err != nil {
		return err
	}

	//Timestamp
	err = writeVarInt(w, pver, msg.Timestamp)
	if err != nil {
		return err
	}

	//Signature
	err = writeVarBytes(w, uint32(ed25519.SignatureSize), msg.Sig)
	if err != nil {
		return err
	}

	return nil
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgCommitEntry) BtcDecode(r io.Reader, pver uint32) error {
	//ECPubKey
	bytes, err := readVarBytes(r, pver, uint32(notaryapi.HashSize), CmdCommitEntry)
	if err != nil {
		return err
	}

	msg.ECPubKey = new(notaryapi.Hash)
	msg.ECPubKey.SetBytes(bytes)

	//EntryHash
	bytes, err = readVarBytes(r, pver, uint32(notaryapi.HashSize), CmdCommitEntry)
	if err != nil {
		return err
	}
	msg.EntryHash = new(notaryapi.Hash)
	msg.EntryHash.SetBytes(bytes)

	//Credits
	err = readElement(r, &msg.Credits) // change it to varint??
	if err != nil {
		return err
	}

	//Timestamp
	msg.Timestamp, err = readVarInt(r, pver)
	if err != nil {
		return err
	}

	//Signature
	msg.Sig, err = readVarBytes(r, pver, uint32(ed25519.SignatureSize), CmdCommitEntry)
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

// NewMsgInv returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgCommitEntry() *MsgCommitEntry {
	return &MsgCommitEntry{}
}
