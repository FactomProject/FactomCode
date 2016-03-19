package common

import (
)

type BroadcastChannel struct {
	Channels []chan interface{}
}

func (bc *BroadcastChannel) AddChannel (c chan interface{}) bool {
	if c == nil {
		//Shouldn't add nil channels, they will stall
		return false
	}
	bc.Channels = append(bc.Channels, c)
	return true
}

func (bc *BroadcastChannel) Broadcast(data interface{}) {
	for i:=len(bc.Channels)-1;i>=0;i-- {
		func () {
				defer func() {
						if r := recover(); r != nil {
							//Channel is closed, removing closed channel
							bc.Channels = append(bc.Channels[i:], bc.Channels[:i+1]...)
						}
					}()
					bc.Channels[i]<-data
			}()
	}
}

func (bc *BroadcastChannel) Len() int {
	return len(bc.Channels)
}