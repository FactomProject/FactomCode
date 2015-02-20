// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"github.com/FactomProject/FactomCode/notaryapi"
	"io"
)

//this is the wire format for Factoin tx messages
type MsgTx struct {
	Data []byte
	/*
		Version int32
		TxData 	[]byte
		Sigs	[]byte
	*/
}

func (msg MsgTx) TxType() string {
	return "factoid"
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgTx) BtcEncode(w io.Writer, pver uint32) (err error) {

	/*
		err = writeElement(w, &msg.Version)
		if err != nil {
			return err
		}

		err = writeVarBytes(w, pver, msg.TxData)
		if err != nil {
			return err
		}

		err = writeVarBytes(w, pver, msg.Sigs)
		if err != nil {
			return err
		}
	*/
	err = writeVarBytes(w, pver, msg.Data)
	if err != nil {
		return err
	}

	return nil
}

func (msg *MsgTx) BtcDecode(r io.Reader, pver uint32) (err error) {
	/*
		err = readElement(r, &msg.Version)
		if err != nil {
			return err
		}

		msg.TxData, err = readVarBytes(r, pver, 0, CmdTx)
		if err != nil {
			return err
		}

		msg.Sigs, err = readVarBytes(r, pver, 0, CmdTx)
		if err != nil {
			return err
		}
	*/

	msg.Data, err = readVarBytes(r, pver, 0, CmdTx)
	if err != nil {
		return err
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgTx) Command() string {
	return CmdTx
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgTx) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

// TxMessage is an interface that describes a factom transaction message.
// TxMessage has already been decoded from the wire and is identified as CmdTx.
//
// TxMessage is designed to be used for generic processing of all factom messages
// such factoid, entry=credit,  commit/reveal chain/entry, and confirmations.
//
// Initial implimentation will be for factoid only.
type TxMessage interface {
	notaryapi.BinaryMarshallable
	TxType() string
}
