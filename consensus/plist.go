package consensus

import (
	"github.com/FactomProject/FactomCode/wire"
)

// ProcessList contains a list of valid confirmation messages
// and is used for consensus building
type ProcessList struct {
	plItems    []*ProcessListItem
	nextIndex  int
	totalItems int
}

// ProcessListItem  contains a list of valid confirmation messages
// and is used for consensus building
type ProcessListItem struct {
	Ack     *wire.MsgAck
	Msg     wire.FtmInternalMsg
	MsgHash *wire.ShaHash
}

// NewProcessList creates a new process list
func NewProcessList(sizeHint uint) *ProcessList {
	return &ProcessList{
		plItems:    make([]*ProcessListItem, 0, sizeHint),
		nextIndex:  0,
		totalItems: 0,
	}
}

// AddToProcessList Adds the process list entry in the right slot
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

// GetPLItems Get Process lit items
func (pl *ProcessList) GetPLItems() []*ProcessListItem {
	return pl.plItems
}

// GetMissingMsg Gets Process lit item based on missing msg, and returns either
// ack or msg based on the type of msg missing.
func (pl *ProcessList) GetMissingMsg(msg *wire.MsgMissing) wire.Message {
	for _, item := range pl.plItems {
		if item.Ack.Height == msg.Height && item.Ack.Index == msg.Index {
			if msg.IsEomAck() {
				return item.Ack
			}
			m, ok := item.Msg.(wire.Message)
			if ok {
				return m
			}
		}
	}
	return nil
}
