package common

import (
	"sync"
)

type BroadcastChannel struct {
	Channels []*ChannelBuffer
}

func (bc *BroadcastChannel) NewChannel() chan interface{} {
	buf:=NewChannelBuffer()
	bc.Channels = append(bc.Channels, buf)
	return buf.OutChannel
}

/*
func (bc *BroadcastChannel) AddChannel (c chan interface{}) bool {
	if c == nil {
		//Shouldn't add nil channels, they will stall
		return false
	}
	bc.Channels = append(bc.Channels, c)
	return true
}*/

func (bc *BroadcastChannel) Broadcast(data interface{}) {
	for i:=0;i<len(bc.Channels);i++ {
		bc.Channels[i].InChannel<-data
	}
}

func (bc *BroadcastChannel) Len() int {
	return len(bc.Channels)
}

type ChannelBuffer struct {
	//use this to send data in
	InChannel chan interface{}

	//Internal data
	OutChannel chan interface{}
	Messages []interface{}
	//Semaphores
	NewItemToSend *sync.Cond
	MessageSemaphore *sync.Mutex
}

func NewChannelBuffer() *ChannelBuffer {
	cb:=new(ChannelBuffer)
	cb.InChannel = make(chan interface{}, 128)
	cb.OutChannel = make(chan interface{}, 128)
	cb.Messages = make([]interface{}, 0, 128)
	cb.MessageSemaphore= new(sync.Mutex)
	cb.NewItemToSend = sync.NewCond(new(sync.Mutex))

	go cb.RunInBuffer()
	go cb.RunOutBuffer()

	return cb
}

func (cb *ChannelBuffer) RunInBuffer() {
	for {
		//Wait for data to come in
		data := <- cb.InChannel
		//Lock mutex
		cb.MessageSemaphore.Lock()
		//Store the data to be sent
		cb.Messages = append(cb.Messages, data)
		//Unlock the mutex
		cb.MessageSemaphore.Unlock()
		//Signal new message has arrived
		cb.NewItemToSend.Broadcast()
	}
}

func (cb *ChannelBuffer) RunOutBuffer() {
	var data interface{}
	//Lock the mutex to prevent panics as per package instructions
	cb.NewItemToSend.L.Lock()
	for {
		data = nil
		if len(cb.Messages)==0 {
			//Wait for new data to arrive
			cb.NewItemToSend.Wait()
		}
		//Lock mutex
		cb.MessageSemaphore.Lock()
		//Fetch data from the stack
		data, cb.Messages = cb.Messages[0], cb.Messages[1:]
		//Unlock the mutex
		cb.MessageSemaphore.Unlock()
		//Send data
		cb.OutChannel<-data
	}
}