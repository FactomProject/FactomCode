// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package server

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FactomProject/FactomCode/common"
	cp "github.com/FactomProject/FactomCode/controlpanel"
	"github.com/FactomProject/FactomCode/wire"
	"github.com/davecgh/go-spew/spew"
)

const (
	chanBufferSize = 50

	// minInFlightBlocks is the minimum number of blocks that should be
	// in the request queue for headers-first mode before requesting
	// more.
	minInFlightBlocks = 10

	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

// ackMsg packages a ACK message and the peer it came from together
// so the processor has access to that information.
type ackMsg struct {
	ack  *wire.MsgAck
	peer *peer
}

// dirBlockMsg packages a directory block message and the peer it came from together
// so the block handler has access to that information.
type dirBlockMsg struct {
	block *common.DirectoryBlock
	peer  *peer
}

// dirInvMsg packages a dir block inv message and the peer it came from together
// so the block handler has access to that information.
type dirInvMsg struct {
	inv  *wire.MsgDirInv
	peer *peer
}

// newPeerMsg signifies a newly connected peer to the block handler.
type newPeerMsg struct {
	peer *peer
}

// donePeerMsg signifies a newly disconnected peer to the block handler.
type donePeerMsg struct {
	peer *peer
}

// getSyncPeerMsg is a message type to be sent across the message channel for
// retrieving the current sync peer.
type getSyncPeerMsg struct {
	reply chan *peer
}

// isCurrentMsg is a message type to be sent across the message channel for
// requesting whether or not the block manager believes it is synced with
// the currently connected peers.
type isCurrentMsg struct {
	reply chan bool
}

// pauseMsg is a message type to be sent across the message channel for
// pausing the block manager.  This effectively provides the caller with
// exclusive access over the manager until a receive is performed on the
// unpause channel.
type pauseMsg struct {
	unpause <-chan struct{}
}

// chainState tracks the state of the best chain as blocks are inserted.  This
// is done because btcchain is currently not safe for concurrent access and the
// block manager is typically quite busy processing block and inventory.
// Therefore, requesting this information from chain through the block manager
// would not be anywhere near as efficient as simply updating it as each block
// is inserted and protecting it with a mutex.
type chainState struct {
	sync.Mutex
	newestHash        *wire.ShaHash
	newestHeight      int32
	pastMedianTime    time.Time
	pastMedianTimeErr error
}

// Best returns the block hash and height known for the tip of the best known
// chain.
//
// This function is safe for concurrent access.
func (c *chainState) Best() (*wire.ShaHash, int32) {
	c.Lock()
	defer c.Unlock()

	return c.newestHash, c.newestHeight
}

// blockManager provides a concurrency safe block manager for handling all
// incoming blocks.
type blockManager struct {
	server            *server
	started           int32
	shutdown          int32
	// requestedTxns     map[wire.ShaHash]struct{}
	// requestedBlocks   map[wire.ShaHash]struct{}
	// receivedLogBlocks int64
	// receivedLogTx     int64
	processingReqs    bool
	syncPeer          *peer
	msgChan           chan interface{}
	chainState        chainState
	wg                sync.WaitGroup
	quit              chan struct{}
}

// updateChainState updates the chain state associated with the block manager.
// This allows fast access to chain information since btcchain is currently not
// safe for concurrent access and the block manager is typically quite busy
// processing block and inventory.
func (b *blockManager) updateChainState(newestHash *wire.ShaHash, newestHeight int32) {
	b.chainState.Lock()
	defer b.chainState.Unlock()

	b.chainState.newestHash = newestHash
	b.chainState.newestHeight = newestHeight
}

// handleNewPeerMsg deals with new peers that have signalled they may
// be considered as a sync peer (they have already successfully negotiated).  It
// also starts syncing if needed.  It is invoked from the syncHandler goroutine.
func (b *blockManager) handleNewPeerMsg(peers *list.List, p *peer) {
	// Ignore if in the process of shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	bmgrLog.Infof("New valid peer %s", p)

	// Ignore the peer if it's not a sync candidate.
	if !b.isSyncCandidateFactom(p) {
		return
	}

	// Add the peer as a candidate to sync from.
	peers.PushBack(p)
	bmgrLog.Infof("Total peers after peers.PushBack(%s): %d", p, peers.Len())

	// Start syncing by choosing the best candidate if needed.
	b.startSyncFactom(peers)
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It
// removes the peer as a candidate for syncing and in the case where it was
// the current sync peer, attempts to select a new best peer to sync from.  It
// is invoked from the syncHandler goroutine.
func (b *blockManager) handleDonePeerMsg(peers *list.List, p *peer) {
	// Remove the peer from the list of candidate peers.
	for e := peers.Front(); e != nil; e = e.Next() {
		if e.Value == p {
			peers.Remove(e)
			break
		}
	}

	bmgrLog.Infof("Lost peer %s", p)

	// Remove requested transactions from the global map so that they will
	// be fetched from elsewhere next time we get an inv.
	// for k := range p.requestedTxns {
		// delete(b.requestedTxns, k)
	// }

	// Remove requested blocks from the global map so that they will be
	// fetched from elsewhere next time we get an inv.
	// TODO(oga) we could possibly here check which peers have these blocks
	// and request them now to speed things up a little.
	// for k := range p.requestedBlocks {
		// delete(b.requestedBlocks, k)
	// }

	// Attempt to find a new peer to sync from if the quitting peer is the
	// sync peer.  Also, reset the headers-first state if in headers-first
	// mode so
	if b.syncPeer != nil && b.syncPeer == p {
		b.syncPeer = nil
		b.startSyncFactom(peers)
	}
}

// current returns true if we believe we are synced with our peers, false if we
// still have blocks to check
func (b *blockManager) current() bool {
	// if blockChain thinks we are current and we have no syncPeer it
	// is probably right.
	if b.syncPeer == nil {
		return true
	}

	_, height, err := db.FetchBlockHeightCache() //b.server.db.NewestSha()
	// No matter what chain thinks, if we are below the block we are
	// syncing to we are not current.
	if err != nil || height < int64(b.syncPeer.lastBlock) {
		return false
	}

	return true
}

// haveInventory returns whether or not the inventory represented by the passed
// inventory vector is known.  
func (b *blockManager) haveInventory(invVect *wire.InvVect) (bool, error) {
	switch invVect.Type {

	case wire.InvTypeFactomDirBlock:
		// Ask db if the block is known to it in any form (main
		// chain, side chain, or orphan).
		return b.haveBlockInDB((&invVect.Hash).ToFactomHash())
	}
	// The requested inventory is is an unsupported type, so just claim
	// it is known to avoid requesting it.
	return true, nil
}

// haveBlockInDB returns whether or not the chain instance has the block represented
// by the passed hash.  This includes checking the various places a block can
// be like part of the main chain, on a side chain, or in the orphan pool.
//
// This function is NOT safe for concurrent access.
func (b *blockManager) haveBlockInDB(hash *common.Hash) (bool, error) {
	blk, _ := db.FetchDBlockByHash(hash) 
	// most errors are "leveldb not found"
	// if err != nil {
		// fmt.Println("haveBlockInDB: error in db.FetchDBlockByHash. ", err.Error())
		// return false, err
	// }
	if blk != nil {
		return true, nil
	}
	return false, nil
}

// blockHandler is the main handler for the block manager.  It must be run
// as a goroutine.  It processes block and inv messages in a separate goroutine
// from the peer handlers so the block (MsgBlock) messages are handled by a
// single thread without needing to lock memory data structures.  This is
// important because the block manager controls which blocks are needed and how
// the fetching should proceed.
func (b *blockManager) blockHandler() {
	b.wg.Add(1)
	defer func() {
		b.wg.Done()
	}()

	candidatePeers := list.New()
out:
	for {
		select {
		case m := <-b.msgChan:
			switch msg := m.(type) {
			case *newPeerMsg:
				b.handleNewPeerMsg(candidatePeers, msg.peer)

			case *donePeerMsg:
				b.handleDonePeerMsg(candidatePeers, msg.peer)

			case getSyncPeerMsg:
				msg.reply <- b.syncPeer

			case isCurrentMsg:
				msg.reply <- b.current()

			case pauseMsg:
				// Wait until the sender unpauses the manager.
				<-msg.unpause

			case *dirInvMsg:
				b.handleDirInvMsg(msg)

			case *ackMsg:
				b.handleAckMsg(msg)

			default:
				bmgrLog.Warnf("Invalid message type in block "+
					"handler: %T", msg)
				fmt.Printf("before invalid message type: msg=%s\n", spew.Sdump(msg))
				panic(errors.New("invalid message type"))
			}

		case <-b.quit:
			break out
		}
	}

	bmgrLog.Trace("Block handler done")
}

// NewPeer informs the block manager of a newly active peer.
func (b *blockManager) NewPeer(p *peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &newPeerMsg{peer: p}
}

// DonePeer informs the blockmanager that a peer has disconnected.
func (b *blockManager) DonePeer(p *peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &donePeerMsg{peer: p}
}

// Start begins the core block handler which processes block and inv messages.
func (b *blockManager) Start() {
	// Already started?
	if atomic.AddInt32(&b.started, 1) != 1 {
		return
	}

	bmgrLog.Trace("Starting block manager")

	go b.blockHandler()
}

// Stop gracefully shuts down the block manager by stopping all asynchronous
// handlers and waiting for them to finish.
func (b *blockManager) Stop() error {
	if atomic.AddInt32(&b.shutdown, 1) != 1 {
		bmgrLog.Warnf("Block manager is already in the process of " +
			"shutting down")
		return nil
	}

	bmgrLog.Infof("Block manager shutting down")
	close(b.quit)
	b.wg.Wait()
	return nil
}

// SyncPeer returns the current sync peer.
func (b *blockManager) SyncPeer() *peer {
	reply := make(chan *peer)
	b.msgChan <- getSyncPeerMsg{reply: reply}
	return <-reply
}

// IsCurrent returns whether or not the block manager believes it is synced with
// the connected peers.
func (b *blockManager) IsCurrent() bool {
	reply := make(chan bool)
	b.msgChan <- isCurrentMsg{reply: reply}
	return <-reply
}

// Pause pauses the block manager until the returned channel is closed.
//
// Note that while paused, all peer and block processing is halted.  The
// message sender should avoid pausing the block manager for long durations.
func (b *blockManager) Pause() chan<- struct{} {
	c := make(chan struct{})
	b.msgChan <- pauseMsg{c}
	return c
}

// newBlockManager returns a new factom block manager.
// Use Start to begin processing asynchronous block and inv updates.
func newBlockManager(s *server) (*blockManager, error) {
	//newestHash, height, err := s.db.NewestSha()
	//if err != nil {
	//return nil, err
	//}

	bm := blockManager{
		server:          s,
		// requestedTxns:   make(map[wire.ShaHash]struct{}),
		// requestedBlocks: make(map[wire.ShaHash]struct{}),
		msgChan:         make(chan interface{}, cfg.MaxPeers*3),
		quit:            make(chan struct{}),
	}

	// Initialize the chain state now that the intial block node index has
	// been generated.
	//bm.updateChainState(newestHash, height)

	return &bm, nil
}

// handleDirInvMsg handles dir inv messages from all peers.
// We examine the inventory advertised by the remote peer and act accordingly.
func (b *blockManager) handleDirInvMsg(imsg *dirInvMsg) {
	//bmgrLog.Debug("handleDirInvMsg: ", spew.Sdump(imsg))

	// Attempt to find the final block in the inventory list.  There may
	// not be one.
	lastBlock := -1
	invVects := imsg.inv.InvList
	bmgrLog.Debugf("handleDirInvMsg: len(InvVects)=%d", len(invVects))
	for i := len(invVects) - 1; i >= 0; i-- {
		if invVects[i].Type == wire.InvTypeFactomDirBlock {
			lastBlock = i
			bmgrLog.Debugf("handleDirInvMsg: lastBlock=%d", lastBlock)
			break
		}
	}

	// If this inv contains a block annoucement, and this isn't coming from
	// our current sync peer or we're current, then update the last
	// announced block for this peer. We'll use this information later to
	// update the heights of peers based on blocks we've accepted that they
	// previously announced.
	if lastBlock != -1 && (imsg.peer != b.syncPeer || b.current()) {
		imsg.peer.UpdateLastAnnouncedBlock(&invVects[lastBlock].Hash)
	}

	// Ignore invs from peers that aren't the sync if we are not current.
	// Helps prevent fetching a mass of orphans.
	if imsg.peer != b.syncPeer && !b.current() {
		return
	}

	// If our chain is current and a peer announces a block we already
	// know of, then update their current block height.
	if lastBlock != -1 && b.current() {
		h := &invVects[lastBlock].Hash
		hash, _ := common.NewShaHash(h.Bytes())
		dblock, err := db.FetchDBlockByHash(hash)
		if err == nil && dblock != nil {
			imsg.peer.UpdateLastBlockHeight(int32(dblock.Header.DBHeight))
			bmgrLog.Infof("handleDirInvMsg: UpdateLastBlockHeight: %d, %s, %s",
				dblock.Header.DBHeight, hash.String(), imsg.peer)
		}
	}

	/*dblock, err := db.FetchDBlockByHash((&invVects[lastBlock].Hash).ToFactomHash())
	if err != nil {
		bmgrLog.Warnf("Unable to fetch dir block for (hash: %v), %v",
			(&invVects[lastBlock].Hash).ToFactomHash(), err)
	} else {
		imsg.peer.UpdateLastBlockHeight(int32(dblock.Header.DBHeight))
	}*/

	// Request the advertised inventory if we don't already have it.  
	for i, iv := range invVects {
		// Ignore unsupported inventory types.
		if iv.Type != wire.InvTypeFactomDirBlock { //} && iv.Type != wire.InvTypeTx {
			continue
		}

		// Add the inventory to the cache of known inventory
		// for the peer.
		imsg.peer.AddKnownInventory(iv)

		// Request the inventory if we don't already have it.
		haveInv, err := b.haveInventory(iv)
		if err != nil {
			bmgrLog.Warnf("Unexpected failure when checking for "+
				"existing inventory during inv message "+
				"processing: %v", err)
			continue
		}
		if !haveInv {
			// Add it to the request queue.
			imsg.peer.requestQueue = append(imsg.peer.requestQueue, iv)
			// continue
		}

		// We already have the final block advertised by this
		// inventory message, so force a request for more.  
		if i == lastBlock {
			// Request blocks after this one up to the
			// final one the remote peer knows about (zero
			// stop hash).
			bmgrLog.Debug("push for more dir blocks: PushGetDirBlocksMsg")
			locator := DirBlockLocatorFromHash(&iv.Hash)
			imsg.peer.PushGetDirBlocksMsg(locator, &zeroBtcHash)
		}
	}

	// Request as much as possible at once.  Anything that won't fit into
	// the request will be requested on the next inv message.
	numRequested := 0
	gdmsg := wire.NewMsgGetDirData()
	requestQueue := imsg.peer.requestQueue
	for len(requestQueue) != 0 {
		iv := requestQueue[0]
		requestQueue[0] = nil
		requestQueue = requestQueue[1:]

		switch iv.Type {
		case wire.InvTypeFactomDirBlock:
			// Request the block if there is not already a pending
			// request.
			// if _, exists := b.requestedBlocks[iv.Hash]; !exists {
				// b.requestedBlocks[iv.Hash] = struct{}{}
				// imsg.peer.requestedBlocks[iv.Hash] = struct{}{}
				gdmsg.AddInvVect(iv)
				numRequested++
			// }

			//case wire.InvTypeFactoidTx:  ???
		}

		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	imsg.peer.requestQueue = requestQueue
	if len(gdmsg.InvList) > 0 {
		imsg.peer.QueueMessage(gdmsg, nil)
	}
}

// QueueAck adds the passed Ack message and peer to the block handling queue.
func (b *blockManager) QueueAck(msg *wire.MsgAck, p *peer) {
	// Don't accept more acks if we're shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		p.blockProcessed <- struct{}{}
		return
	}

	b.msgChan <- &ackMsg{ack: msg, peer: p}
}

// QueueDirBlock adds the passed GetDirBlocks message and peer to the block handling queue.
func (b *blockManager) QueueDirBlock(msg *wire.MsgDirBlock, p *peer) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		p.blockProcessed <- struct{}{}
		return
	}

	b.msgChan <- &dirBlockMsg{block: msg.DBlk, peer: p}
}

// QueueDirInv adds the passed inv message and peer to the block handling queue.
func (b *blockManager) QueueDirInv(inv *wire.MsgDirInv, p *peer) {
	// No channel handling here because peers do not need to block on inv
	// messages.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &dirInvMsg{inv: inv, peer: p}
}

// startSyncFactom will choose the best peer among the available candidate peers to
// download/sync the blockchain from.  When syncing is already running, it
// simply returns.  It also examines the candidates for any which are no longer
// candidates and removes them as needed.
func (b *blockManager) startSyncFactom(peers *list.List) {
	// Return now if we're already syncing.
	if b.syncPeer != nil {
		bmgrLog.Info("syncPeer: ", b.syncPeer)
		return
	}

	// Find the height of the current known best block.
	_, height, err := db.FetchBlockHeightCache()
	if err != nil {
		bmgrLog.Errorf("%v", err)
		return
	}

	bmgrLog.Infof("Latest DirBlock Height: %d", height)

	var bestPeer *peer
	var enext *list.Element
	for e := peers.Front(); e != nil; e = enext {
		enext = e.Next()
		p := e.Value.(*peer)

		if p.lastBlock <= int32(height) {
			peers.Remove(e)
			continue
		}

		if p.IsLeader() {
			bestPeer = p
			break
		} 

		if bestPeer == nil || bestPeer != nil && p.lastBlock > bestPeer.lastBlock {
			bestPeer = p
		}
	}
	bmgrLog.Info("bestPeer: ", bestPeer)

	// Start syncing from the best peer if one was selected.
	if bestPeer != nil {
		locator, err := LatestDirBlockLocator()
		if err != nil {
			bmgrLog.Errorf("Failed to get block locator for the "+
				"latest block: %v", err)
			return
		}

		bmgrLog.Infof("LatestDirBlockLocator: %s", spew.Sdump(locator))

		str := fmt.Sprintf("At %d: syncing to block height %d from peer %v",
			height, bestPeer.lastBlock, bestPeer.addr)
		bmgrLog.Infof(str)
		
		cp.CP.AddUpdate(
			"Syncing", // tag
			"status",  // Category
			"Client is Syncing with Federated Server(s)", // Title
			str, // Message
			60)
		
		bestPeer.PushGetDirBlocksMsg(locator, &zeroBtcHash)
		b.syncPeer = bestPeer
		if common.SERVER_NODE == b.server.nodeType {
			fed := b.server.GetMyFederateServer()
			fed.FirstJoined = uint32(bestPeer.lastBlock) + 1
			fmt.Println("BlckchainManager: startSyncFactom: my FirstJoined=", fed.FirstJoined)
		}
	} else {
		bmgrLog.Warnf("No sync peer candidates available")
	}
}

// isSyncCandidateFactom returns whether or not the peer is a candidate to consider
// syncing from.
func (b *blockManager) isSyncCandidateFactom(p *peer) bool {
	// If i'm a server, and peer is a NodeCandidate (including client), 
	// then, this peer is not a sync candidate.
	// Todo: We should have a "NodeClient" as a NodeState, 
	// so that client is able to download blocks from other clients
	if common.SERVER_NODE == factomConfig.App.NodeMode && p.IsCandidate() {
		bmgrLog.Infof("peer is not a sync candidate: ", p)
		return false
	}
	return true
}

// handleAckMsg handles ACK messages from all peers.
func (b *blockManager) handleAckMsg(amsg *ackMsg) {
	if b.server.IsLeader() {
		return
	}
	// let EOM10 work as a heartbeat for clients to know real height
	// and check if they are missing blocks
	if  ClientOnly && amsg.ack.Type != wire.EndMinute10 {
		return
	}

	bytes, err := amsg.ack.GetBinaryForSignature()
	if err != nil {
		fmt.Println("blockManager.handleAckMsg: error in GetBinaryForSignature: ", err.Error())
		return
	}

	// todo: use sourceNodeID's peer, not amsg.peer, to verify sig
	if !amsg.peer.pubKey.Verify(bytes, &amsg.ack.Signature) {
		fmt.Println("blockManager.handleAckMsg: Invalid signature in ", amsg.ack)
		// return
	}

	missingMsgs, err := processAckMsg(amsg.ack)
	if err != nil {
		fmt.Printf("blockManager.handleAckMsg: %s, err=%s\n", amsg.ack, err.Error())
		return
	}
	// todo: use InvVest to send it in one msg
	for _, m := range missingMsgs {
		amsg.peer.QueueMessage(m, nil)
	}
}
