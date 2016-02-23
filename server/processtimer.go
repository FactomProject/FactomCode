// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"time"

	"github.com/FactomProject/FactomCode/wire"
)

// BlockTimer is set to sent End-Of-Minute messages to processor
type BlockTimer struct {
	nextDBlockHeight uint32
	inMsgQueue       chan wire.FtmInternalMsg //incoming message queue for factom control messages
}

// Send End-Of-Minute messages to processor for the current open directory block
func (bt *BlockTimer) StartBlockTimer() {

	//wait till the end of minute
	//the first minute section might be bigger than others. To be improved.
	/*	t := time.Now()
		time.Sleep(time.Duration((60 - t.Second()) * 1000000000))
	*/
	fmt.Println("BlockTimer.StartBlockTimer. nextDBlockHeight=", bt.nextDBlockHeight)
	if directoryBlockInSeconds < 600 {
		sleeptime := directoryBlockInSeconds / 10

		// Set the start time for the open dir block
		dchain.NextBlock.Header.Timestamp = uint32(time.Now().Round(time.Minute).Unix() / 60)

		for i := 0; i < 10; i++ {
			eomMsg := &wire.MsgInt_EOM{
				EOM_Type:         wire.END_MINUTE_1 + byte(i),
				NextDBlockHeight: bt.nextDBlockHeight,
			}

			//send the end-of-minute message to processor
			bt.inMsgQueue <- eomMsg

			time.Sleep(time.Duration(sleeptime * 1000000000))
		}
		return
	}

	roundTime := time.Now().Round(time.Minute)
	minutesPassed := roundTime.Minute() - (roundTime.Minute()/10)*10

	// Set the start time for the open dir block
	dchain.NextBlock.Header.Timestamp = uint32(roundTime.Add(time.Duration((0-60*minutesPassed)*1000000000)).Unix() / 60)

	for minutesPassed < 10 {

		// Sleep till the end of minute
		t0 := time.Now()
		t0_round := t0.Round(time.Minute)
		if t0.Before(t0_round) {
			time.Sleep(time.Duration((60 + t0.Second()) * 1000000000))
		} else {
			time.Sleep(time.Duration((60 - t0.Second()) * 1000000000))
		}

		eomMsg := &wire.MsgInt_EOM{
			EOM_Type:         wire.END_MINUTE_1 + byte(minutesPassed),
			NextDBlockHeight: bt.nextDBlockHeight,
		}

		//send the end-of-minute message to processor
		bt.inMsgQueue <- eomMsg

		minutesPassed++
	}

}
