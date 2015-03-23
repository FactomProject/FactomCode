package consensus

import (
	"github.com/FactomProject/btcd/wire"
	//"fmt"
)

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessList struct {
	plItems    []*ProcessListItem
	nextIndex  int
	totalItems int
}

// Process list contains a list of valid confirmation messages
// and is used for consensus building
type ProcessListItem struct {
	Ack     *wire.MsgAcknowledgement
	Msg     wire.FtmInternalMsg
	MsgHash *wire.ShaHash
}

// create a new process list
func NewProcessList(sizeHint uint) *ProcessList {

	return &ProcessList{
		plItems:    make([]*ProcessListItem, 0, sizeHint),
		nextIndex:  0,
		totalItems: 0,
	}
}

// Add the process list entry in the right slot
func (pl *ProcessList) AddToProcessList(pli *ProcessListItem) error {

	// Increase the slice capacity if needed
	if pli.Ack.Index >= uint32(cap(pl.plItems)) {
		temp := make([]*ProcessListItem, len(pl.plItems), pli.Ack.Index*2)
		copy(temp, pl.plItems)
		pl.plItems = temp
	}

	// Increase the slice length if needed
	if pli.Ack.Index >= uint32(len(pl.plItems)) {
		pl.plItems = pl.plItems[0 : pli.Ack.Index+1]
	}

	pl.plItems[pli.Ack.Index] = pli

	return nil
}

// Validate the process list
func (pl *ProcessList) IsValid() bool {

	// to add logic ??
	return true
}

// Get Process lit items
func (pl *ProcessList) GetPLItems() []*ProcessListItem {
	return pl.plItems
}
