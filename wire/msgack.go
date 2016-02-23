// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/FactomProject/FactomCode/common"
)

// Ack Type
const (
	ACK_FACTOID_TX uint8 = iota
	END_MINUTE_1
	END_MINUTE_2
	END_MINUTE_3
	END_MINUTE_4
	END_MINUTE_5
	END_MINUTE_6
	END_MINUTE_7
	END_MINUTE_8
	END_MINUTE_9
	END_MINUTE_10
	ACK_REVEAL_ENTRY
	ACK_COMMIT_CHAIN
	ACK_REVEAL_CHAIN
	ACK_COMMIT_ENTRY

	FORCE_FACTOID_GENESIS_REBUILD
	//	FORCE_FACTOID_VALIDATION // at a specific block height; everything higher will be discarded by btcd-side (possibly creating orphaned blocks)
	INFO_CURRENT_HEIGHT // info message to the wire-side to indicate the current known block height; a duplicate of FORCE_FACTOID_VALIDATION (???)
)

type MsgAck struct {
	Height      uint32
	ChainID     *common.Hash
	Index       uint32
	Type        byte
	Affirmation *ShaHash // affirmation value -- hash of the message/object in question
	SerialHash  [32]byte
	Signature   [64]byte
}

// Write out the MsgAck (excluding Signature) to binary.
func (msg *MsgAck) GetBinaryForSignature() (data []byte, err error) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg.Height)
	if msg.ChainID != nil {
		data, err = msg.ChainID.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	binary.Write(&buf, binary.BigEndian, msg.Index)
	buf.Write([]byte{msg.Type})
	buf.Write(msg.Affirmation.Bytes())
	buf.Write(msg.SerialHash[:])
	return buf.Bytes(), err
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgAck) BtcDecode(r io.Reader, pver uint32) error {
	//err := readElements(r, &msg.Height, msg.ChainID, &msg.Index, &msg.Type, msg.Affirmation, &msg.SerialHash, &msg.Signature)
	newData, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("MsgAck.BtcDecode reader is invalid")
	}
	if len(newData) != 169 {
		return fmt.Errorf("MsgAck.BtcDecode reader does not have right length: ", len(newData))
	}

	msg.Height, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]

	msg.ChainID = common.NewHash()
	newData, _ = msg.ChainID.UnmarshalBinaryData(newData)

	msg.Index, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]
	msg.Type, newData = newData[0], newData[1:]
	msg.Affirmation, _ = NewShaHash(newData[:32])

	newData = newData[32:]
	copy(msg.SerialHash[:], newData[0:32])
	newData = newData[32:]
	copy(msg.Signature[:], newData[0:64])
	return nil
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgAck) BtcEncode(w io.Writer, pver uint32) error {
	//err := writeElements(w, msg.Height, msg.ChainID, msg.Index, msg.Type, msg.Affirmation, msg.SerialHash, msg.Signature)
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg.Height)
	buf.Write(msg.ChainID.Bytes())
	binary.Write(&buf, binary.BigEndian, msg.Index)
	buf.WriteByte(msg.Type)
	buf.Write(msg.Affirmation.Bytes())
	buf.Write(msg.SerialHash[:])
	buf.Write(msg.Signature[:])
	w.Write(buf.Bytes())
	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgAck) Command() string {
	return CmdAck
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgAck) MaxPayloadLength(pver uint32) uint32 {
	// 10K is too big of course, TODO: adjust
	return MaxAppMsgPayload
}

// NewMsgAck returns a new bitcoin ping message that conforms to the Message
// interface.  See MsgAck for details.
func NewMsgAck(height uint32, index uint32, affirm *ShaHash, ackType byte) *MsgAck {
	if affirm == nil {
		affirm = new(ShaHash)
	}
	return &MsgAck{
		Height:      height,
		ChainID:     common.NewHash(), //TODO: get the correct chain id from processor
		Index:       index,
		Affirmation: affirm,
		Type:        ackType,
	}
}

// Sha() Creates a sha hash from the message binary (output of BtcEncode)
func (msg *MsgAck) Sha() (ShaHash, error) {
	buf := bytes.NewBuffer(nil)
	msg.BtcEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))
	return sha, nil
}

func (msg *MsgAck) Clone() *MsgAck {
	return &MsgAck{
		Height:      msg.Height,
		ChainID:     msg.ChainID,
		Index:       msg.Index,
		Affirmation: msg.Affirmation,
		Type:        msg.Type,
	}
}

func (msg *MsgAck) IsEomAck() bool {
	if END_MINUTE_1 <= msg.Type && msg.Type <= END_MINUTE_10 {
		return true
	}
	return false
}
