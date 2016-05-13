// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgRevealChain implements the Message interface and represents a factom
// Reveal-Chain message.  It is used by client to reveal the chain.
type MsgRevealChain struct {
	FirstEntry *common.Entry
}

// MsgEncode encodes the receiver to w using the factom protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgRevealChain) MsgEncode(w io.Writer, pver uint32) error {

	//FirstEntry
	bytes, err := msg.FirstEntry.MarshalBinary()
	if err != nil {
		return err
	}

	err = writeVarBytes(w, pver, bytes)
	if err != nil {
		return err
	}

	return nil
}

// MsgDecode decodes r using the factom protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgRevealChain) MsgDecode(r io.Reader, pver uint32) error {
	//FirstEntry
	bytes, err := readVarBytes(r, pver, MaxAppMsgPayload, CmdRevealChain)
	if err != nil {
		return err
	}

	msg.FirstEntry = new(common.Entry)
	err = msg.FirstEntry.UnmarshalBinary(bytes)
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

// NewMsgRevealChain returns a new .MsgRevealChain
func NewMsgRevealChain() *MsgRevealChain {
	return &MsgRevealChain{}
}

// Sha Creates a sha hash from the message binary (output of MsgEncode)
func (msg *MsgRevealChain) Sha() (ShaHash, error) {

	buf := bytes.NewBuffer(nil)
	msg.MsgEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))

	return sha, nil
}

// String returns its shows representation
func (msg *MsgRevealChain) String() string {
	return fmt.Sprintf("MsgRevealChain: entry.chainID: %s", msg.FirstEntry.ChainID.String())
}
