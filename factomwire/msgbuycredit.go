// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"io"
	"github.com/FactomProject/FactomCode/notaryapi"	
)


// This is a temp msg. It will be replaced by a factoid transaction.
type MsgBuyCredit struct {
	ECPubKey *notaryapi.Hash
	FactoidBase uint64
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgBuyCredit) BtcEncode(w io.Writer, pver uint32) error {
	
	//ECPubKey	
	err := writeVarBytes(w, uint32(notaryapi.HashSize), msg.ECPubKey.Bytes)
	if err != nil {
		return err
	}
	
	//FactoidBase	
	err = writeVarInt(w, pver, msg.FactoidBase)
	if err != nil {
		return err
	}
	
	return nil
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgBuyCredit) BtcDecode(r io.Reader, pver uint32) error {
	bytes, err := readVarBytes(r, pver, uint32(notaryapi.HashSize), CmdBuyCredit)
	if err != nil {
		return err
	}

	//ECPubKey
	msg.ECPubKey = new (notaryapi.Hash)
	msg.ECPubKey.SetBytes(bytes)
	
	//FactoidBase
	msg.FactoidBase, err = readVarInt(r, pver)
		
	if err != nil {
		return err
	}
	
	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgBuyCredit) Command() string {
	return CmdBuyCredit
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgBuyCredit) MaxPayloadLength(pver uint32) uint32 {
	return MaxAppMsgPayload
}

// NewMsgInv returns a new bitcoin inv message that conforms to the Message
// interface.  See MsgInv for details.
func NewMsgBuyCredit() *MsgBuyCredit {
	return &MsgBuyCredit{
	}
}


