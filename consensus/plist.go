package consensus

import (
	"github.com/FactomProject/btcd/wire"
)

// Orphan process list map to hold our of order confirmation messages
// key: MsgAcknowledgement.MsgHash.String()
var OrphanPLMap map[string]*wire.MsgAcknowledgement 

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessList struct {
	 PlEntries []*wire.MsgAcknowledgement // to add a flag??	
	 DirBlkHeight uint64 
	 // fed server id??
}

// create a new process list 
func NewProcessList(sizeHint uint, dirBlkHeight uint64) *ProcessList {

	return &ProcessList{
		PlEntries: make([]*wire.MsgAcknowledgement, 0, sizeHint),
		DirBlkHeight: dirBlkHeight,
	}
}
 
// Add the process list entry in the right slot
func (pl *ProcessList) AddToProcessList(msg *wire.MsgAcknowledgement) error {

	// Increase the slice capacity if needed
	if msg.Index >= uint32(cap(pl.PlEntries)){
			temp := make([]*wire.MsgAcknowledgement, len(pl.PlEntries), (cap(pl.PlEntries)+1)*2)
			copy(temp, pl.PlEntries)
			pl.PlEntries = temp	
	}
	
	// Increase the slice length if needed		
	if msg.Index >= uint32(len(pl.PlEntries)){	
		pl.PlEntries = pl.PlEntries[0:msg.Index+1]
	}
	
	pl.PlEntries[msg.Index] = msg
 
	return nil
}

// Initialize the process list from the orphan process list map
// Out of order Ack messages are stored in OrphanPLMap
func (pl *ProcessList) InitProcessListFromOrphanMap() error {

	for key, msgAck := range OrphanPLMap {
		if pl.DirBlkHeight == msgAck.Height{
			pl.AddToProcessList(msgAck)
			delete(OrphanPLMap, key)
		}

	}
 
	return nil
}

