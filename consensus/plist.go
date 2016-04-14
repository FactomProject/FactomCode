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

// GetPLItems Get Process list items
func (pl *ProcessList) GetPLItems() []*ProcessListItem {
	return pl.plItems
}

// GetNextIndex Get Process list nextIndex
func (pl *ProcessList) GetNextIndex() int {
	return pl.nextIndex
}

// GetMissingMsg Gets Process lit item based on missing msg, and returns either
// ack or msg based on the type of msg missing.
func (pl *ProcessList) GetMissingMsg(msg *wire.MsgMissing) wire.Message {
	for _, item := range pl.plItems {
		// todo: what if missing both msg and ack for the same height and index?
		if item.Ack.Height == msg.Height && item.Ack.Index == msg.Index {
			if item.Ack.IsEomAck() || msg.IsAck {
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

// GetEndMinuteAck return end of minute ack based on eom type provided
func (pl *ProcessList) GetEndMinuteAck(EOMType byte) *wire.MsgAck {
	for _, item := range pl.plItems {
		if item != nil && item.Ack != nil && item.Ack.Type == EOMType {
			return item.Ack
		}
	}
	return nil
}

// GetCoinbaseTimestamp return the CoinbaseTimestampt`for this process list
func (pl *ProcessList) GetCoinbaseTimestamp() uint64 {
	for _, item := range pl.plItems {
		if item != nil && item.Ack != nil && item.Ack.IsEomAck() {
			return item.Ack.CoinbaseTimestamp
		}
	}
	return 0
}
