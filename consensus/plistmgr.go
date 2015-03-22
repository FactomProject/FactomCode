package consensus

import (
	"errors"
	"sync"	
	"github.com/FactomProject/btcd/wire"
)

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessListMgr struct {
	sync.RWMutex	
	MyProcessList     *ProcessList
	OtherProcessLists []*ProcessList

	NextDBlockHeight uint64

	// Orphan process list map to hold our of order confirmation messages
	// key: MsgAcknowledgement.MsgHash.String()
	OrphanPLMap map[string]*ProcessListItem
}

// create a new process list
func NewProcessListMgr(height uint64, otherPLSize int, plSizeHint uint) *ProcessListMgr {

	plMgr := new(ProcessListMgr)
	plMgr.MyProcessList = NewProcessList(plSizeHint)
	plMgr.OtherProcessLists = make([]*ProcessList, otherPLSize, otherPLSize)
	for i := 0; i < len(plMgr.OtherProcessLists); i++ {
		plMgr.OtherProcessLists[i] = NewProcessList(plSizeHint)
	}
	plMgr.NextDBlockHeight = height

	return plMgr
}

// Add a ProcessListItem into the corresponding process list
func (plMgr *ProcessListMgr) AddToProcessList(plItem *ProcessListItem) error {

	// If the item belongs to my process list
	if plItem.ack == nil {
		plMgr.AddToMyProcessList(plItem)
	} else {
		plMgr.AddToOtherProcessList(plItem)
	}

	return nil
}

//Added to OtherPL[0] - to be improved after milestone 1??
func (plMgr *ProcessListMgr) AddToOtherProcessList(plItem *ProcessListItem) error {
	// Determin which process list to add
	plMgr.OtherProcessLists[0].AddToProcessList(plItem)
	return nil
}

//Added to OtherPL[0] - to be improved after milestone 1??
func (plMgr *ProcessListMgr) AddToOrphanProcessList(plItem *ProcessListItem) error {
	// Determin which process list to add
	//	plMgr.OrphanPLMap[string(plItem.ack.Affirmation)] = plItem
	return nil
}

// Add a factom transaction to the my process list
// Each of the federated servers has one MyProcessList
func (plMgr *ProcessListMgr) AddToMyProcessList(plItem *ProcessListItem) error {

	if wire.CmdTx == plItem.msg.Command() {
		_, ok := plItem.msg.(*wire.MsgTx)
		if ok {
			// Come up with the right process list index for the new item
			index := uint32(len(plMgr.MyProcessList.plItems))
			if index > 0 {
				lastPlItem := plMgr.MyProcessList.plItems[index-1]
				if lastPlItem.ack == nil {
					return errors.New("Invalid process list.")
				}
				if index != lastPlItem.ack.Index+1 {
					return errors.New("Invalid process list index.")
				}
			}
			msgAck := wire.NewMsgAcknowledgement(plMgr.NextDBlockHeight, index, plItem.msgHash, wire.ACK_FACTOID_TX)
			//msgAck.Affirmation = plItem.msgHash.Bytes

			plItem.ack = msgAck

			//Add the item into my process list
			plMgr.MyProcessList.plItems[index] = plItem

		}
	}

	//Broadcast the plitem into the network??

	return nil
}

// Initialize the process list from the orphan process list map
// Out of order Ack messages are stored in OrphanPLMap
func (plMgr *ProcessListMgr) InitProcessListFromOrphanMap() error {

	for key, plItem := range plMgr.OrphanPLMap {
		if plMgr.NextDBlockHeight == plItem.ack.Height {
			plMgr.MyProcessList.AddToProcessList(plItem)
			delete(plMgr.OrphanPLMap, key)
		}

	}

	return nil
}

// Create a new process list item and add it to the MyProcessList
func (plMgr *ProcessListMgr) AddProcessListItem(msg wire.FtmInternalMsg, hash *wire.ShaHash, msgType byte) error {
	plMgr.Lock()
	ack := wire.NewMsgAcknowledgement(plMgr.NextDBlockHeight, uint32(plMgr.MyProcessList.nextIndex), hash, msgType)
	plMgr.MyProcessList.nextIndex++
	plMgr.Unlock()
	
	plItem := &ProcessListItem{
		ack:	ack,
		msg:	msg,
		msgHash:hash,
		}
	plMgr.AddToProcessList(plItem)
		
	return nil
}