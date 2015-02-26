// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"sync"
	"sync/atomic"

	"github.com/FactomProject/FactomCode/factomwire"
)

// txMsg packages a bitcoin tx message and the peer it came from together
// so the block handler has access to that information.
type txMsg struct {
	tx   *factomwire.MsgTx
	peer *peer
}

type confMsg struct {
	conf *factomwire.MsgConfirmation
	peer *peer
}

type blockManager struct {
	server   *server
	started  int32
	shutdown int32
	msgChan  chan interface{}
	wg       sync.WaitGroup
	quit     chan struct{}
}

// newBlockManager returns a new bitcoin block manager.
// Use Start to begin processing asynchronous block and inv updates.
func newBlockManager(s *server) (*blockManager, error) {

	bm := blockManager{
		server:  s,
		msgChan: make(chan interface{}),
		quit:    make(chan struct{}),
	}

	return &bm, nil
}

// Start begins the core block handler which processes block and inv messages.
func (b *blockManager) Start() {
	// Already started?
	if atomic.AddInt32(&b.started, 1) != 1 {
		return
	}

	//bmgrLog.Trace("Starting block manager")
	b.wg.Add(1)
	go b.blockHandler()
}

// Stop gracefully shuts down the block manager by stopping all asynchronous
// handlers and waiting for them to finish.
func (b *blockManager) Stop() error {
	if atomic.AddInt32(&b.shutdown, 1) != 1 {
		//bmgrLog.Warnf("Block manager is already in the process of " +
		//	"shutting down")
		return nil
	}

	//bmgrLog.Infof("Block manager shutting down")
	close(b.quit)
	b.wg.Wait()
	return nil
}

// QueueTx adds the passed transaction message and peer to the block handling
// queue.
func (b *blockManager) QueueTx(msg *factomwire.MsgTx, p *peer) {
	// Don't accept more transactions if we're shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		p.txProcessed <- struct{}{}
		return
	}

	b.msgChan <- &txMsg{tx: msg, peer: p}
}

// QueueTx adds the passed transaction message and peer to the block handling
// queue.
func (b *blockManager) QueueConf(msg *factomwire.MsgConfirmation, p *peer) {
	// Don't accept more transactions if we're shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		p.txProcessed <- struct{}{}
		return
	}

	b.msgChan <- &confMsg{conf: msg, peer: p}
}
func (b *blockManager) handleTxMsg(tmsg *txMsg) {
	if b.server.txMemPool.ProcessTransaction(tmsg.tx) {
		b.server.BroadcastMessage(tmsg.tx, tmsg.peer)
	}
}

func (b *blockManager) handleMsgConfirmation(msgc *confMsg) {
	if b.server.txMemPool.ProcessConfirmation(msgc.conf) {
		b.server.BroadcastMessage(msgc.conf, msgc.peer)
	}
}

func (b *blockManager) blockHandler() {
out:
	for {
		select {
		case m := <-b.msgChan:
			switch msg := m.(type) {

			case *txMsg:
				b.handleTxMsg(msg)
				if msg.peer != nil {
					msg.peer.txProcessed <- struct{}{}
				}
			case *confMsg:
				b.handleMsgConfirmation(msg)
			}

		case <-b.quit:
			break out
		}
	}
	b.wg.Done()
}
