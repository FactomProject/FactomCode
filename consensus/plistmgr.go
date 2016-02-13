package consensus

import (
	"sync"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/btcd/wire"
)

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessListMgr struct {
	sync.RWMutex
	MyProcessList    *ProcessList
	NextDBlockHeight uint32
	serverPrivKey    common.PrivateKey
}

// create a new process list
func NewProcessListMgr(height uint32, otherPLSize int, plSizeHint uint, privKey common.PrivateKey) *ProcessListMgr {
	plMgr := new(ProcessListMgr)
	plMgr.MyProcessList = NewProcessList(plSizeHint)
	plMgr.NextDBlockHeight = height
	plMgr.serverPrivKey = privKey
	return plMgr
}

// Create a new process list item and add it to the MyProcessList
func (plMgr *ProcessListMgr) AddToFollowersProcessList(msg wire.FtmInternalMsg, ack *wire.MsgAck) error {
	return nil
}

// Create a new process list item and add it to the MyProcessList
func (plMgr *ProcessListMgr) AddToLeadersProcessList(msg wire.FtmInternalMsg, hash *wire.ShaHash, msgType byte) (ack *wire.MsgAck, err error) {
	ack = wire.NewMsgAck(plMgr.NextDBlockHeight, uint32(plMgr.MyProcessList.nextIndex), hash, msgType)
	// Sign the ack using server private keys
	bytes, _ := ack.GetBinaryForSignature()
	ack.Signature = *plMgr.SignAck(bytes).Sig
	plMgr.MyProcessList.nextIndex++

	plItem := &ProcessListItem{
		Ack:     ack,
		Msg:     msg,
		MsgHash: hash,
	}
	plMgr.MyProcessList.AddToProcessList(plItem)
	return ack, nil
}

// Sign the Ack --
//TODO: to be moved into util package
func (plMgr *ProcessListMgr) SignAck(bytes []byte) (sig common.Signature) {
	sig = plMgr.serverPrivKey.Sign(bytes)
	return sig
}

// Check if the number of process list items is exceeding the size limit
func (plMgr *ProcessListMgr) IsMyPListExceedingLimit() bool {
	return (plMgr.MyProcessList.totalItems >= common.MAX_PLIST_SIZE)
}
