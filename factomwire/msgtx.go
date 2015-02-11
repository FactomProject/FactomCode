// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"io"
	//"github.com/FactomProject/FactomCode/notaryapi"	
)

//this is the wire format for Factoin tx messages
type MsgTx struct {
	Data 	[]byte
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgTx) BtcEncode(w io.Writer, pver uint32) error {

	err := writeVarBytes(w, pver, msg.Data)
	if err != nil {
		return err
	}
	
	return nil
}

func (msg *MsgTx) BtcDecode(r io.Reader, pver uint32) (err error) {
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