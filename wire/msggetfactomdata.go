
package wire

import (
	"fmt"
	"io"
)

// MsgGetFactomData implements the Message interface and represents a factom
// getfactomdata message.  It is used to request data such as blocks and entries
// from another peer, based on height. Each message is limited to a maximum number of
// inventory vectors, which is currently 50,000.  As a result, multiple messages
// must be used to request larger amounts of data.
//
// Use the AddInvVectHeight function to build up the list of inventory vectors when
// sending a getfactomdata message to another peer.
type MsgGetFactomData struct {
	InvList []*InvVectHeight
}

// AddInvVectHeight adds an inventory vector to the message.
func (msg *MsgGetFactomData) AddInvVectHeight(iv *InvVectHeight) error {
	if len(msg.InvList)+1 > MaxInvPerMsg {
		str := fmt.Sprintf("too many InvVectHeight in message [max %v]",
			MaxInvPerMsg)
		return messageError("MsgGetFactomData.AddInvVectHeight", str)
	}

	msg.InvList = append(msg.InvList, iv)
	return nil
}

// MsgDecode decodes r using the protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetFactomData) MsgDecode(r io.Reader, pver uint32) error {
	count, err := readVarInt(r, pver)
	if err != nil {
		return err
	}

	// Limit to max inventory vectors per message.
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many InvVectHeight in message [%v]", count)
		return messageError("MsgGetFactomData.MsgDecode", str)
	}

	msg.InvList = make([]*InvVectHeight, 0, count)
	for i := uint64(0); i < count; i++ {
		iv := InvVectHeight{}
		err := readInvVectHeight(r, pver, &iv)
		if err != nil {
			return err
		}
		msg.AddInvVectHeight(&iv)
	}

	return nil
}

// MsgEncode encodes the receiver to w using the protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetFactomData) MsgEncode(w io.Writer, pver uint32) error {
	// Limit to max inventory vectors per message.
	count := len(msg.InvList)
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many InvVectHeight in message [%v]", count)
		return messageError("MsgGetFactomData.MsgEncode", str)
	}

	err := writeVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, iv := range msg.InvList {
		err := writeInvVectHeight(w, pver, iv)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetFactomData) Command() string {
	return CmdGetFactomData
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetFactomData) MaxPayloadLength(pver uint32) uint32 {
	// Num inventory vectors (varInt) + max allowed inventory vectors.
	return MaxVarIntPayload + (MaxInvPerMsg * maxInvVectPayload)
}

// NewMsgGetFactomData returns a new bitcoin getdata message that conforms to the
// Message interface.  See MsgGetFactomData for details.
func NewMsgGetFactomData() *MsgGetFactomData {
	return &MsgGetFactomData{
		InvList: make([]*InvVectHeight, 0, defaultInvListAlloc),
	}
}

// NewMsgGetFactomDataSizeHint returns a new factom getdata message that conforms to
// the Message interface.  
func NewMsgGetFactomDataSizeHint(sizeHint uint) *MsgGetFactomData {
	// Limit the specified hint to the maximum allow per message.
	if sizeHint > MaxInvPerMsg {
		sizeHint = MaxInvPerMsg
	}

	return &MsgGetFactomData{
		InvList: make([]*InvVectHeight, 0, sizeHint),
	}
}