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

// StartBlockTimer sends End-Of-Minute messages to processor for the current open directory block
func (bt *BlockTimer) StartBlockTimer() {

	//wait till the end of minute
	//the first minute section might be bigger than others. To be improved.
	fmt.Println("BlockTimer.StartBlockTimer. nextDBlockHeight=", bt.nextDBlockHeight)
	if directoryBlockInSeconds == 60 {

		// Set the start time for the open dir block in case not being set
		// it's set when leader crashes and a new leader inherited a process list.
		if dchain.NextBlock.Header.Timestamp == 0 {
			dchain.NextBlock.Header.Timestamp = uint32(time.Now().Round(time.Minute).Unix() / 60)
		}
		
		sleeptime := directoryBlockInSeconds / 10
		for i := 0; i < 10; i++ {
			eomMsg := &wire.MsgInt_EOM{
				EOM_Type:         wire.EndMinute1 + byte(i),
				NextDBlockHeight: bt.nextDBlockHeight,
			}
			//send the end-of-minute message to processor
			bt.inMsgQueue <- eomMsg
			time.Sleep(time.Duration(sleeptime * 1000000000))
		}
		return
	}

	if directoryBlockInSeconds == 600 {
		n := time.Now()
		minutesPassed := n.Minute() % 10

		// Set the start time for the open dir block
		if dchain.NextBlock.Header.Timestamp == 0 {
			// dchain.NextBlock.Header.Timestamp = uint32(roundTime.Add(time.Duration((0-60*minutesPassed)*1000000000)).Unix() / 60)
			dchain.NextBlock.Header.Timestamp = uint32(n.Truncate(time.Minute).Unix() / 60)
		}

		fmt.Printf("*BlockTimer: minutesPassed=%d, ts=%d, now=%s\n", minutesPassed, 
			dchain.NextBlock.Header.Timestamp, n)
		
		for minutesPassed < 10 {

			// Sleep till the end of minute
			time.Sleep(time.Duration(60 - time.Now().Second()) * time.Second)
			fmt.Printf("BlockTimer: minutesPassed=%d, now=%s\n", minutesPassed, time.Now())

			eomMsg := &wire.MsgInt_EOM{
				EOM_Type:         wire.EndMinute1 + byte(minutesPassed),
				NextDBlockHeight: bt.nextDBlockHeight,
			}

			bt.inMsgQueue <- eomMsg
			minutesPassed++
		}
	}
}


// startBlockTimer is a new timer run by both leader and followers
func startBlockTimer() {	
	var ts uint32
	if directoryBlockInSeconds == 60 {
		i := 0
		n0 := time.Now()
		time.Sleep(time.Duration(1000000000 - n0.Nanosecond()) * time.Nanosecond)
		n1 := time.Now()
		time.Sleep(time.Duration(60 - n1.Second()) * time.Second)
		ts = uint32(time.Now().Truncate(time.Minute).Unix() / 60)
		fmt.Println("\nnow0=", n0, ", now1=", n1, ", i=", i, ", ts=", dchain.NextBlock.Header.Timestamp)

		c := time.Tick(6 * time.Second)
		blockTicker(i, ts, c)
	}

	if directoryBlockInSeconds == 600 {
		n0 := time.Now()
		time.Sleep(time.Duration(1000000000 - n0.Nanosecond()) * time.Nanosecond)
		n1 := time.Now()
		time.Sleep(time.Duration(60 - n1.Second()) * time.Second)
		n2 := time.Now()
		i := n2.Minute()
		if i >= 10 {
			i = i % 10
		} 
		
		// if there's only 1 or 2 min left, sleep it out
		if i > 7 {
			time.Sleep(time.Duration(10 - i) * time.Minute)
			i = time.Now().Minute() + 1
			if i >= 10 {
				i = i % 10
			} 
		}

		ts = uint32(time.Now().Truncate(time.Minute).Unix() / 60)
		fmt.Println("\nnow0=", n0, ", now1=", n1, ", now2=", n2, ", i=", i, ", ts=", dchain.NextBlock.Header.Timestamp)

		c := time.Tick(1 * time.Minute)
		blockTicker(i, ts, c)
	}
}

func blockTicker(i int, ts uint32, c <-chan time.Time) {
	for now := range c {
		if i < 10 {
			i++
		} else {
			i = 1
		}

		if i == 10 {
			ts = uint32(time.Now().Truncate(time.Minute).Unix() / 60)
		}
		if i == 1 && localServer.IsLeader() {
			dchain.NextBlock.Header.Timestamp = ts
		}
		fmt.Printf("\ntimer: h=%d type=%d ts=%d %v\n", dchain.NextDBHeight, i, dchain.NextBlock.Header.Timestamp, now)
		
		if localServer.IsLeader() {
			eom := &wire.MsgInt_EOM{
				EOM_Type:         byte(i),
				NextDBlockHeight: dchain.NextDBHeight,
			}
			processLeaderEOM(eom)
		}
	}
}