// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/FactomProject/FactomCode/common"
)

// MsgMissing is used to request missing msg, ack and blocks during or after
// process list and building blocks
type MsgMissing struct {
	Height    uint32  //DBHeight for this process list
	Index     uint32  //offset in this process list
	Type      byte    //See Ack / msg types and InvTypes of blocks
	IsAck     bool    //yes means this missing msg is an ack, otherwise it's a msg
	ShaHash   ShaHash //shahash of the msg if IsAck is false
	ReqNodeID string  // requestor's nodeID
	Sig       common.Signature
}

// GetBinaryForSignature Writes out the MsgMissing (excluding Signature) to binary.
func (msg *MsgMissing) GetBinaryForSignature() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg.Height)
	binary.Write(&buf, binary.BigEndian, msg.Index)
	buf.WriteByte(msg.Type)
	var b byte
	if msg.IsAck {
		b = 1
	}
	buf.WriteByte(b)
	buf.Write(msg.ShaHash.Bytes())
	buf.Write([]byte(msg.ReqNodeID))
	return buf.Bytes()
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgMissing) MsgDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgMissing.MsgDecode reader is not a " + "*bytes.Buffer")
	}
	err := readElements(buf, &msg.Height, &msg.Index, &msg.Type, &msg.IsAck, &msg.ShaHash, &msg.Sig)
	if err != nil {
		return err
	}
	if buf.Len() > 0 {
		nodeID, err := readVarString(buf, pver)
		if err != nil {
			return err
		}
		msg.ReqNodeID = nodeID
	}
	return nil
	/*
		newData, err := ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("MsgMissing.MsgDecode reader is invalid")
		}

		msg.Height, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]
		msg.Index, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]
		msg.Type = newData[0]
		//msg.Type, newData = newData[0], newData[1:]
		//len := len(newData) - 96
		//msg.ReqNodeID = string(newData[:len])
		//msg.Sig = common.UnmarshalBinarySignature(newData[len:])
		return nil
	*/
}

// MsgEncode is part of the Message interface implementation.
func (msg *MsgMissing) MsgEncode(w io.Writer, pver uint32) error {
	err := writeElements(w, msg.Height, msg.Index, msg.Type, msg.IsAck, msg.ShaHash, msg.Sig)
	if err != nil {
		fmt.Errorf(err.Error())
		return err
	}
	err = writeVarString(w, pver, msg.ReqNodeID)
	if err != nil {
		return err
	}
	return nil
	/*
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, msg.Height)
		binary.Write(&buf, binary.BigEndian, msg.Index)
		buf.WriteByte(msg.Type)
		//buf.Write([]byte(msg.ReqNodeID))
		//data := common.MarshalBinarySignature(msg.Sig)
		//buf.Write(data[:])
		w.Write(buf.Bytes())
		return nil*/
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgMissing) Command() string {
	return CmdMissing
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgMissing) MaxPayloadLength(pver uint32) uint32 {
	return 255
}

// NewMsgMissing returns a new factom ping message that conforms to the Message
// interface.  See MsgMissing for details.
func NewMsgMissing(height uint32, index uint32, typ byte, isAck bool,
	shaHash ShaHash, sourceID string) *MsgMissing {
	return &MsgMissing{
		Height:    height,
		Index:     index,
		Type:      typ,
		IsAck:     isAck,
		ShaHash:   shaHash,
		ReqNodeID: sourceID,
	}
}

// Sha Creates a sha hash from the message binary (output of MsgEncode)
func (msg *MsgMissing) Sha() (ShaHash, error) {
	buf := bytes.NewBuffer(nil)
	msg.MsgEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))
	return sha, nil
}

// IsEomAck checks if it's a EOM ack
func (msg *MsgMissing) IsEomAck() bool {
	if EndMinute1 <= msg.Type && msg.Type <= EndMinute10 {
		return true
	}
	return false
}

// Equals check if two MsgMissings are the same
func (msg *MsgMissing) Equals(ack *MsgMissing) bool {
	return msg.Height == ack.Height &&
		msg.Index == ack.Index &&
		msg.Type == ack.Type &&
		msg.IsAck == ack.IsAck &&
		bytes.Compare(msg.ShaHash.Bytes(), ack.ShaHash.Bytes()) == 0 &&
		msg.ReqNodeID == ack.ReqNodeID &&
		msg.Sig.Equals(ack.Sig)
}

// ByMsgIndex sorts MsgMissing by its Index
type ByMsgIndex []*MsgMissing

func (s ByMsgIndex) Len() int {
	return len(s)
}
func (s ByMsgIndex) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByMsgIndex) Less(i, j int) bool {
	return s[i].Index < s[j].Index
}
