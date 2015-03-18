package consensus

import (
	"github.com/FactomProject/btcd/wire"
	//"fmt"
)

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessList struct {
	plItems []*ProcessListItem
}

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessListItem struct {
	ack     *wire.MsgAcknowledgement
	msg     wire.Message
	msgHash *wire.ShaHash
}

// create a new process list
func NewProcessList(sizeHint uint) *ProcessList {

	return &ProcessList{
		plItems: make([]*ProcessListItem, 0, sizeHint),
	}
}

// Add the process list entry in the right slot
func (pl *ProcessList) AddToProcessList(pli *ProcessListItem) error {

	// Increase the slice capacity if needed
	if pli.ack.Index >= uint32(cap(pl.plItems)) {
		temp := make([]*ProcessListItem, len(pl.plItems), pli.ack.Index*2)
		copy(temp, pl.plItems)
		pl.plItems = temp
	}

	// Increase the slice length if needed
	if pli.ack.Index >= uint32(len(pl.plItems)) {
		pl.plItems = pl.plItems[0 : pli.ack.Index+1]
	}

	pl.plItems[pli.ack.Index] = pli

	return nil
}
