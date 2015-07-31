package consensus

import (
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/btcd/wire"
	"sync"
)

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessListMgr struct {
	sync.RWMutex
	MyProcessList     *ProcessList
	OtherProcessLists []*ProcessList

	NextDBlockHeight uint32
	//Server Private key and Public key for milestone 1
	serverPrivKey common.PrivateKey

	// Orphan process list map to hold our of order confirmation messages
	// key: MsgAcknowledgement.MsgHash.String()
	OrphanPLMap map[string]*ProcessListItem
}

// create a new process list
func NewProcessListMgr(height uint32, otherPLSize int, plSizeHint uint, privKey common.PrivateKey) *ProcessListMgr {

	plMgr := new(ProcessListMgr)
	plMgr.MyProcessList = NewProcessList(plSizeHint)
	plMgr.OtherProcessLists = make([]*ProcessList, otherPLSize, otherPLSize)
	for i := 0; i < len(plMgr.OtherProcessLists); i++ {
		plMgr.OtherProcessLists[i] = NewProcessList(plSizeHint)
	}
	plMgr.NextDBlockHeight = height
	plMgr.serverPrivKey = privKey

	return plMgr
}

// Add a ProcessListItem into the corresponding process list
/*func (plMgr *ProcessListMgr) AddToProcessList(plItem *ProcessListItem) error {

	// If the item belongs to my process list
	if plItem.Ack == nil {
		plMgr.AddToMyProcessList(plItem)
	} else {
		plMgr.AddToOtherProcessList(plItem)
	}

	return nil
}*/

//TODO: Added to OtherPL[0] - to be improved after milestone 1
func (plMgr *ProcessListMgr) AddToOtherProcessList(plItem *ProcessListItem) error {
	// Determin which process list to add
	plMgr.OtherProcessLists[0].AddToProcessList(plItem)
	return nil
}

//TODO: Added to OtherPL[0] - to be improved after milestone 1
func (plMgr *ProcessListMgr) AddToOrphanProcessList(plItem *ProcessListItem) error {
	// Determin which process list to add
	//	plMgr.OrphanPLMap[string(plItem.ack.Affirmation)] = plItem
	return nil
}

// Add a factom transaction to the my process list
// Each of the federated servers has one MyProcessList
/*func (plMgr *ProcessListMgr) AddToMyProcessList(plItem *ProcessListItem, msgType byte) error {

	// Come up with the right process list index for the new item
	index := uint32(len(plMgr.MyProcessList.plItems))
	if index > 0 {
		lastPlItem := plMgr.MyProcessList.plItems[index-1]
		if lastPlItem.Ack == nil {
			return errors.New("Invalid process list.")
		}
		if index != lastPlItem.Ack.Index+1 {
			return errors.New("Invalid process list index.")
		}
	}
	msgAck := wire.NewMsgAcknowledgement(plMgr.NextDBlockHeight, index, plItem.MsgHash, msgType)

	//msgAck.Affirmation = plItem.msgHash.Bytes
	plItem.Ack = msgAck

	//Add the item into my process list
	plMgr.MyProcessList.AddToProcessList(plItem)

	//Broadcast the plitem into the network??

	return nil
}
*/
// Initialize the process list from the orphan process list map
// Out of order Ack messages are stored in OrphanPLMap
func (plMgr *ProcessListMgr) InitProcessListFromOrphanMap() error {

	for key, plItem := range plMgr.OrphanPLMap {
		if plMgr.NextDBlockHeight == plItem.Ack.Height {
			plMgr.MyProcessList.AddToProcessList(plItem)
			delete(plMgr.OrphanPLMap, key)
		}

	}

	return nil
}

// Create a new process list item and add it to the MyProcessList
func (plMgr *ProcessListMgr) AddMyProcessListItem(msg wire.FtmInternalMsg, hash *wire.ShaHash, msgType byte) (ack *wire.MsgAcknowledgement, err error) {

	ack = wire.NewMsgAcknowledgement(plMgr.NextDBlockHeight, uint32(plMgr.MyProcessList.nextIndex), hash, msgType)
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
