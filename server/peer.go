// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"container/list"
	"encoding/hex"
	"fmt"
	"io"
	prand "math/rand"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FactomProject/factoid"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/server/addrmgr"
	"github.com/FactomProject/FactomCode/wire"
	"github.com/FactomProject/go-socks/socks"
	"github.com/davecgh/go-spew/spew"
)

const (
	// ProtocolVersion is the version for webservices, and the limit should not be the version
	// anyway.
	ProtocolVersion = 1005 // version starts from 1000 for Factom

	// maxProtocolVersion is the max protocol version the peer supports.
	maxProtocolVersion = ProtocolVersion

	// outputBufferSize is the number of elements the output channels use.
	outputBufferSize = 50

	// invTrickleSize is the maximum amount of inventory to send in a single
	// message when trickling inventory to remote peers.
	maxInvTrickleSize = 1000

	// maxKnownInventory is the maximum number of items to keep in the known
	// inventory cache.
	maxKnownInventory = 1000

	// negotiateTimeoutSeconds is the number of seconds of inactivity before
	// we timeout a peer that hasn't completed the initial version
	// negotiation.
	negotiateTimeoutSeconds = 30

	// idleTimeoutMinutes is the number of minutes of inactivity before
	// we time out a peer.
	idleTimeoutMinutes = 5

	// pingTimeoutMinutes is the number of minutes since we last sent a
	// message requiring a reply before we will ping a host.
	pingTimeoutMinutes = 2

	// connectionRetryInterval is the base amount of time to wait in between
	// retries when connecting to persistent peers.  It is adjusted by the
	// number of retries such that there is a retry backoff.
	connectionRetryInterval = time.Second * 10

	// maxConnectionRetryInterval is the max amount of time retrying of a
	// persistent peer is allowed to grow to.  This is necessary since the
	// retry logic uses a backoff mechanism which increases the interval
	// base done the number of retries that have been done.
	maxConnectionRetryInterval = time.Minute * 5
)

var (
	// nodeCount is the total number of peer connections made since startup
	// and is used to assign an id to a peer.
	nodeCount int32

	// userAgentName is the user agent name and is used to help identify
	// ourselves to other factom peers.
	userAgentName = "factomd"

	// userAgentVersion is the user agent version and is used to help
	// identify ourselves to other factom peers.
	userAgentVersion = fmt.Sprintf("%d.%d.%d", appMajor, appMinor, appPatch)
)

// zeroBtcHash is the zero value hash (all zeros).  It is defined as a convenience.
var zeroBtcHash wire.ShaHash

// minUint32 is a helper function to return the minimum of two uint32s.
// This avoids a math import and the need to cast to floats.
func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// newNetAddress attempts to extract the IP address and port from the passed
// net.Addr interface and create a factom NetAddress structure using that
// information.
func newNetAddress(addr net.Addr, services wire.ServiceFlag) (*wire.NetAddress, error) {
	// addr will be a net.TCPAddr when not using a proxy.
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		ip := tcpAddr.IP
		port := uint16(tcpAddr.Port)
		na := wire.NewNetAddressIPPort(ip, port, services)
		return na, nil
	}

	// addr will be a socks.ProxiedAddr when using a proxy.
	if proxiedAddr, ok := addr.(*socks.ProxiedAddr); ok {
		ip := net.ParseIP(proxiedAddr.Host)
		if ip == nil {
			ip = net.ParseIP("0.0.0.0")
		}
		port := uint16(proxiedAddr.Port)
		na := wire.NewNetAddressIPPort(ip, port, services)
		return na, nil
	}

	// For the most part, addr should be one of the two above cases, but
	// to be safe, fall back to trying to parse the information from the
	// address string as a last resort.
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	na := wire.NewNetAddressIPPort(ip, uint16(port), services)
	return na, nil
}

// outMsg is used to house a message to be sent along with a channel to signal
// when the message has been sent (or won't be sent due to things such as
// shutdown)
type outMsg struct {
	msg      wire.Message
	doneChan chan struct{}
}

// peer provides a factom peer for handling factom communications.  The
// overall data flow is split into 3 goroutines and a separate block manager.
// Inbound messages are read via the inHandler goroutine and generally
// dispatched to their own handler.  For inbound data-related messages such as
// blocks, transactions, and inventory, the data is passed on to the block
// manager to handle it.  Outbound messages are queued via QueueMessage or
// QueueInventory.  QueueMessage is intended for all messages, including
// responses to data such as blocks and transactions.  QueueInventory, on the
// other hand, is only intended for relaying inventory as it employs a trickling
// mechanism to batch the inventory together.  The data flow for outbound
// messages uses two goroutines, queueHandler and outHandler.  The first,
// queueHandler, is used as a way for external entities (mainly block manager)
// to queue messages quickly regardless of whether the peer is currently
// sending or not.  It acts as the traffic cop between the external world and
// the actual goroutine which writes to the network socket.  In addition, the
// peer contains several functions which are of the form pushX, that are used
// to push messages to the peer.  Internally they use QueueMessage.
type peer struct {
	server     *server
	fctnet     wire.FactomNet
	started    int32
	connected  int32
	disconnect int32 // only to be used atomically
	conn       net.Conn
	addr       string
	na         *wire.NetAddress
	id         int32

	nodeType      string //nodeMode
	nodeID        string //*wire.ShaHash
	pubKey        common.PublicKey
	startTime	  int64
	nodeState	  wire.NodeState

	inbound            bool
	persistent         bool
	knownAddresses     map[string]struct{}
	knownInventory     *MruInventoryMap
	knownInvMutex      sync.Mutex
	// requestedTxns      map[wire.ShaHash]struct{} // owned by blockmanager
	// requestedBlocks    map[wire.ShaHash]struct{} // owned by blockmanager
	retryCount         int64
	prevGetBlocksBegin *wire.ShaHash // owned by blockmanager
	prevGetBlocksStop  *wire.ShaHash // owned by blockmanager
	prevGetHdrsBegin   *wire.ShaHash // owned by blockmanager
	prevGetHdrsStop    *wire.ShaHash // owned by blockmanager
	requestQueue       []*wire.InvVect
	relayMtx           sync.Mutex
	disableRelayTx     bool
	continueHash       *wire.ShaHash
	outputQueue        chan outMsg
	sendQueue          chan outMsg
	sendDoneQueue      chan struct{}
	queueWg            sync.WaitGroup // TODO(oga) wg -> single use channel?
	outputInvChan      chan *wire.InvVect
	txProcessed        chan struct{}
	blockProcessed     chan struct{}
	quit               chan struct{}
	StatsMtx           sync.Mutex // protects all statistics below here.
	versionKnown       bool
	protocolVersion    uint32
	versionSent        bool
	verAckReceived     bool
	services           wire.ServiceFlag
	timeOffset         int64
	timeConnected      time.Time
	lastSend           time.Time
	lastRecv           time.Time
	bytesReceived      uint64
	bytesSent          uint64
	userAgent          string
	startingHeight     int32
	lastBlock          int32
	lastAnnouncedBlock *wire.ShaHash
	lastPingNonce      uint64    // Set to nonce if we have a pending ping.
	lastPingTime       time.Time // Time we sent last ping.
	lastPingMicros     int64     // Time for last ping to return.
}

// String returns the peer's address and directionality as a human-readable
// string.
func (p *peer) String() string {
	return fmt.Sprintf("%s (%s); %s, state=%v", 
		p.addr, directionString(p.inbound), p.nodeID, p.nodeState)
}

// isKnownInventory returns whether or not the peer is known to have the passed
// inventory.  It is safe for concurrent access.
func (p *peer) isKnownInventory(invVect *wire.InvVect) bool {
	p.knownInvMutex.Lock()
	defer p.knownInvMutex.Unlock()

	if p.knownInventory.Exists(invVect) {
		return true
	}
	return false
}

// UpdateLastBlockHeight updates the last known block for the peer. It is safe
// for concurrent access.
func (p *peer) UpdateLastBlockHeight(newHeight int32) {
	p.StatsMtx.Lock()
	defer p.StatsMtx.Unlock()

	peerLog.Tracef("Updating last block height of peer %v from %v to %v",
		p.addr, p.lastBlock, newHeight)
	p.lastBlock = int32(newHeight)
}

// UpdateLastAnnouncedBlock updates meta-data about the last block sha this
// peer is known to have announced. It is safe for concurrent access.
func (p *peer) UpdateLastAnnouncedBlock(blkSha *wire.ShaHash) {
	p.StatsMtx.Lock()
	defer p.StatsMtx.Unlock()

	peerLog.Tracef("Updating last blk for peer %v, %v", p.addr, blkSha)
	p.lastAnnouncedBlock = blkSha
}

// AddKnownInventory adds the passed inventory to the cache of known inventory
// for the peer.  It is safe for concurrent access.
func (p *peer) AddKnownInventory(invVect *wire.InvVect) {
	p.knownInvMutex.Lock()
	defer p.knownInvMutex.Unlock()

	p.knownInventory.Add(invVect)
}

// VersionKnown returns the whether or not the version of a peer is known locally.
// It is safe for concurrent access.
func (p *peer) VersionKnown() bool {
	p.StatsMtx.Lock()
	defer p.StatsMtx.Unlock()

	return p.versionKnown
}

// ProtocolVersion returns the peer protocol version in a manner that is safe
// for concurrent access.
func (p *peer) ProtocolVersion() uint32 {
	p.StatsMtx.Lock()
	defer p.StatsMtx.Unlock()

	return p.protocolVersion
}

// RelayTxDisabled returns whether or not relaying of transactions is disabled.
// It is safe for concurrent access.
func (p *peer) RelayTxDisabled() bool {
	p.relayMtx.Lock()
	defer p.relayMtx.Unlock()

	return p.disableRelayTx
}

// pushVersionMsg sends a version message to the connected peer using the
// current state.
func (p *peer) pushVersionMsg() error {
	_, blockNum, err := db.FetchBlockHeightCache()
	if err != nil {
		return err
	}

	theirNa := p.na

	// If we are behind a proxy and the connection comes from the proxy then
	// we return an unroutable address as their address. This is to prevent
	// leaking the tor proxy address.
	if cfg.Proxy != "" {
		proxyaddress, _, err := net.SplitHostPort(cfg.Proxy)
		// invalid proxy means poorly configured, be on the safe side.
		if err != nil || p.na.IP.String() == proxyaddress {
			theirNa = &wire.NetAddress{
				Timestamp: time.Now(),
				IP:        net.IP([]byte{0, 0, 0, 0}),
			}
		}
	}

	// Version message.
	msg := wire.NewMsgVersion(
		p.server.addrManager.GetBestLocalAddress(p.na), theirNa,
		p.server.nonce, int32(blockNum))
	msg.AddUserAgent(userAgentName, userAgentVersion)

	// XXX: factomd appears to always enable the full node services flag
	// of the remote peer netaddress field in the version message regardless
	// of whether it knows it supports it or not.  Also, factomd sets
	// the services field of the local peer to 0 regardless of support.
	//
	// Realistically, this should be set as follows:
	// - For outgoing connections:
	//    - Set the local netaddress services to what the local peer
	//      actually supports
	//    - Set the remote netaddress services to 0 to indicate no services
	//      as they are still unknown
	// - For incoming connections:
	//    - Set the local netaddress services to what the local peer
	//      actually supports
	//    - Set the remote netaddress services to the what was advertised by
	//      by the remote peer in its version message
	msg.AddrYou.Services = wire.SFNodeNetwork

	// Advertise that we're a full node.
	msg.Services = wire.SFNodeNetwork

	// Advertise our max supported protocol version.
	msg.ProtocolVersion = maxProtocolVersion

	msg.StartTime = p.server.startTime
	msg.NodeType = p.server.nodeType
	msg.NodeID = p.server.nodeID
	msg.NodeSig = p.server.privKey.Sign([]byte(p.server.nodeID))
	msg.SetPassphrase(factomConfig.App.Passphrase)
	
	if !ClientOnly {
		msg.NodeState = p.server.GetMyFederateServer().Peer.nodeState		
	} else {
		msg.NodeState = wire.NodeCandidate
	}

	p.QueueMessage(msg, nil)
	return nil
}

// updateAddresses potentially adds addresses to the address manager and
// requests known addresses from the remote peer depending on whether the peer
// is an inbound or outbound peer and other factors such as address routability
// and the negotiated protocol version.
func (p *peer) updateAddresses(msg *wire.MsgVersion) {
	// Outbound connections.
	if !p.inbound {
		// TODO(davec): Only do this if not doing the initial block
		// download and the local address is routable.
		if !cfg.DisableListen /* && isCurrent? */ {
			// Get address that best matches.
			lna := p.server.addrManager.GetBestLocalAddress(p.na)
			if addrmgr.IsRoutable(lna) {
				addresses := []*wire.NetAddress{lna}
				p.pushAddrMsg(addresses)
			}
		}

		// Request known addresses if the server address manager needs
		// more and the peer has a protocol version new enough to
		// include a timestamp with addresses.
		hasTimestamp := p.ProtocolVersion() >=
			wire.NetAddressTimeVersion
		if p.server.addrManager.NeedMoreAddresses() && hasTimestamp {
			p.QueueMessage(wire.NewMsgGetAddr(), nil)
		}

		// Mark the address as a known good address.
		p.server.addrManager.Good(p.na)
	} else {
		// A peer might not be advertising the same address that it
		// actually connected from.  One example of why this can happen
		// is with NAT.  Only add the address to the address manager if
		// the addresses agree.
		if addrmgr.NetAddressKey(&msg.AddrMe) == addrmgr.NetAddressKey(p.na) {
			p.server.addrManager.AddAddress(p.na, p.na)
			p.server.addrManager.Good(p.na)
		}
	}
}

// handleVersionMsg is invoked when a peer receives a version factom message
// and is used to negotiate the protocol version details as well as kick start
// the communications.
func (p *peer) handleVersionMsg(msg *wire.MsgVersion) {
	fmt.Printf("handleVersionMsg: %s (%s, %d): %s\n", 
		msg.NodeID, msg.NodeType, msg.NodeState, msg.AddrMe.String())
	
	// Detect self connections.
	if msg.Nonce == p.server.nonce {
		peerLog.Debugf("Disconnecting peer connected to self %s", p)
		p.Disconnect()
		return
	}

	if ClientOnly {
		if isVersionMismatch(maxProtocolVersion, msg.ProtocolVersion) {
			errmsg := "\n\n******************** - IMPORTANT - ****************************\n\n"
			errmsg += "\n\n      VERSION MISMATCH -- Please upgrade your software! \n\n"
			errmsg += "\n\n***************************************************************\n\n"
			peerLog.Error(errmsg)
			p.Disconnect()
			fmt.Println(errmsg)
			os.Exit(1)
			//return
		}
	}

	// Updating a bunch of stats.
	p.StatsMtx.Lock()

	// Limit to one version message per peer.
	if p.versionKnown {
		p.logError("Only one version message per peer is allowed %s.",
			p)
		p.StatsMtx.Unlock()

		// Send an reject message indicating the version message was
		// incorrectly sent twice and wait for the message to be sent
		// before disconnecting.
		p.PushRejectMsg(msg.Command(), wire.RejectDuplicate,
			"duplicate version message", nil, true)

		p.Disconnect()
		return
	}

	// Negotiate the protocol version.
	p.protocolVersion = minUint32(p.protocolVersion, uint32(msg.ProtocolVersion))
	p.versionKnown = true
	// peerLog.Debugf("Negotiated protocol version %d for peer %s",
		// p.protocolVersion, p)
	p.lastBlock = msg.LastBlock
	p.startingHeight = msg.LastBlock

	// Set the supported services for the peer to what the remote peer
	// advertised.
	p.services = msg.Services

	// Set the remote peer's user agent.
	p.userAgent = msg.UserAgent

	// Set the peer's time offset.
	p.timeOffset = msg.Timestamp.Unix() - time.Now().Unix()

	// Set the peer's ID.
	p.id = atomic.AddInt32(&nodeCount, 1)

	p.StatsMtx.Unlock()

	// Choose whether or not to relay transactions before a filter command
	// is received.
	p.relayMtx.Lock()
	p.disableRelayTx = msg.DisableRelayTx
	p.relayMtx.Unlock()

	// peerLog.Info("NodeType: ", msg.NodeType, ", NodeID: ", msg.NodeID)
	// only verify id/sig for federate servers
	if common.SERVER_NODE == msg.NodeType {
		if !msg.NodeSig.Verify([]byte(msg.NodeID)) {
			fmt.Println("Error in verifying peer signature: ", p)
			p.PushRejectMsg(msg.Command(), wire.RejectInvalidSig, "invalid signature", nil, true)
			p.Disconnect()
			return
		}
		// not be able to tell who is the existing leader or newly joined leader
		// if msg.NodeState == wire.NodeLeader && p.IsLeader() {
			// panic("I can NOT join as a leader, since there's already a leader in " + msg.NodeID)
		// }

		// Only check passphrase only if both you and I are SERVER node.
		if common.SERVER_NODE == p.server.nodeType && !msg.ComparePassphrase(factomConfig.App.Passphrase) {
			fmt.Println("Error in compare Passphrase: ", msg.Passphrase)
			p.PushRejectMsg(msg.Command(), wire.RejectInvalidPWd, "invalid Passphrase", nil, true)
			p.Disconnect()
			return
		}
		var fed *federateServer
		for _, fed = range p.server.federateServers {
			// Usually when a server has both listen and connect, it has 2 peers (inboud + outbound)
			// this prevents duplication.
			if fed.Peer.nodeID == msg.NodeID {
				fmt.Printf("duplicated fed server / peer: msg.nodeID=%s; peer=%s\n", msg.NodeID, fed.Peer)
				p.Shutdown()
				return
			}
		}
		var h uint32
		if msg.NodeState != wire.NodeCandidate {	// -1 for msg.LastBlock of incoming peer
			h = uint32(msg.LastBlock) + 1
		} else if !p.server.IsCandidate() {
			_, newestHeight, _ := db.FetchBlockHeightCache()	
			h = uint32(newestHeight + 1)
		}
		fedServer := &federateServer{
			Peer:        p,
			FirstJoined: h, 	//todo: it's not right when both peers are candidates
			StartTime:	 msg.StartTime,
		}
		p.server.federateServers = append(p.server.federateServers, fedServer)
		peerLog.Debugf("Signature & Passphrase verified successfully total=%d after adding a new federate server: %s",
			p.server.FederateServerCount(), p)
	} else {
		p.server.clientPeers = append(p.server.clientPeers, p)
	}

	p.startTime = msg.StartTime
	p.nodeState = msg.NodeState
	p.nodeType = msg.NodeType
	p.nodeID = msg.NodeID
	p.pubKey = msg.NodeSig.Pub
	// peerLog.Info("set peer id/type info: ", p)

	// Inbound connections.
	if p.inbound {
		// Set up a NetAddress for the peer to be used with AddrManager.
		// We only do this inbound because outbound set this up
		// at connection time and no point recomputing.
		na, err := newNetAddress(p.conn.RemoteAddr(), p.services)
		if err != nil {
			p.logError("Can't get remote address: %v", err)
			p.Disconnect()
			return
		}
		p.na = na

		// Send version.
		err = p.pushVersionMsg()
		if err != nil {
			p.logError("Can't send version message to %s: %v",
				p, err)
			p.Disconnect()
			return
		}
	}

	// Send verack.
	p.QueueMessage(wire.NewMsgVerAck(), nil)

	// Update the address manager and request known addresses from the
	// remote peer for outbound connections.  This is skipped when running
	// on the simulation test network since it is only intended to connect
	// to specified peers and actively avoids advertising and connecting to
	// discovered peers.
	if !cfg.SimNet {
		p.updateAddresses(msg)
	}

	// Add the remote peer time as a sample for creating an offset against
	// the local clock to keep the network time in sync.
	//p.server.timeSource.AddTimeSample(p.addr, msg.Timestamp)

	// Signal the block manager this peer is a new sync candidate.
	p.server.blockManager.NewPeer(p)

	// TODO: Relay alerts.

	if !ClientOnly {
		// Protocol version mismatch -- need client upgrade !
		if isVersionMismatch(maxProtocolVersion, int32(p.ProtocolVersion())) {
			//util.Trace(fmt.Sprintf("NEED client upgrade -- will ban & disconnect !: us=%d , peer= %d", maxProtocolVersion, p.ProtocolVersion()))
			p.logError(fmt.Sprintf("NEED client upgrade -- will ban & disconnect !: us=%d , peer= %d", maxProtocolVersion, p.ProtocolVersion()))
			//		p.Disconnect()
			p.server.BanPeer(p)
			return
		}
	}
}

// PushRejectMsg sends a reject message for the provided command, reject code,
// and reject reason, and hash.  The hash will only be used when the command
// is a tx or block and should be nil in other cases.  The wait parameter will
// cause the function to block until the reject message has actually been sent.
func (p *peer) PushRejectMsg(command string, code wire.RejectCode, reason string, hash *wire.ShaHash, wait bool) {
	// Don't bother sending the reject message if the protocol version
	// is too low.
	if p.VersionKnown() && p.ProtocolVersion() < wire.RejectVersion {
		return
	}

	msg := wire.NewMsgReject(command, code, reason)
	/*	if command == wire.CmdTx || command == wire.CmdBlock {
		if hash == nil {
			peerLog.Warnf("Sending a reject message for command "+
				"type %v which should have specified a hash "+
				"but does not", command)
			hash = &zeroBtcHash
		}
		msg.Hash = *hash
	}*/

	// Send the message without waiting if the caller has not requested it.
	if !wait {
		p.QueueMessage(msg, nil)
		return
	}

	// Send the message and block until it has been sent before returning.
	doneChan := make(chan struct{}, 1)
	p.QueueMessage(msg, doneChan)
	<-doneChan
}

// handleInvMsg is invoked when a peer receives an inv factom message and is
// used to examine the inventory being advertised by the remote peer and react
// accordingly.  We pass the message down to blockmanager which will call
// QueueMessage with any appropriate responses.
//func (p *peer) handleInvMsg(msg *wire.MsgInv) {
//p.server.blockManager.QueueInv(msg, p)
//}

// handleGetAddrMsg is invoked when a peer receives a getaddr factom message
// and is used to provide the peer with known addresses from the address
// manager.
func (p *peer) handleGetAddrMsg(msg *wire.MsgGetAddr) {
	// Don't return any addresses when running on the simulation test
	// network.  This helps prevent the network from becoming another
	// public test network since it will not be able to learn about other
	// peers that have not specifically been provided.
	if cfg.SimNet {
		return
	}

	// Do not accept getaddr requests from outbound peers.  This reduces
	// fingerprinting attacks.
	if !p.inbound {
		return
	}

	// Get the current known addresses from the address manager.
	addrCache := p.server.addrManager.AddressCache()

	// Push the addresses.
	err := p.pushAddrMsg(addrCache)
	if err != nil {
		p.logError("Can't push address message to %s: %v", p, err)
		p.Disconnect()
		return
	}
}

// pushAddrMsg sends one, or more, addr message(s) to the connected peer using
// the provided addresses.
func (p *peer) pushAddrMsg(addresses []*wire.NetAddress) error {
	// Nothing to send.
	if len(addresses) == 0 {
		return nil
	}

	r := prand.New(prand.NewSource(time.Now().UnixNano()))
	numAdded := 0
	msg := wire.NewMsgAddr()
	for _, na := range addresses {
		// Filter addresses the peer already knows about.
		if _, exists := p.knownAddresses[addrmgr.NetAddressKey(na)]; exists {
			continue
		}

		// If the maxAddrs limit has been reached, randomize the list
		// with the remaining addresses.
		if numAdded == wire.MaxAddrPerMsg {
			msg.AddrList[r.Intn(wire.MaxAddrPerMsg)] = na
			continue
		}

		// Add the address to the message.
		err := msg.AddAddress(na)
		if err != nil {
			return err
		}
		numAdded++
	}
	if numAdded > 0 {
		for _, na := range msg.AddrList {
			// Add address to known addresses for this peer.
			p.knownAddresses[addrmgr.NetAddressKey(na)] = struct{}{}
		}

		p.QueueMessage(msg, nil)
	}
	return nil
}

// handleAddrMsg is invoked when a peer receives an addr factom message and
// is used to notify the server about advertised addresses.
func (p *peer) handleAddrMsg(msg *wire.MsgAddr) {
	// Ignore addresses when running on the simulation test network.  This
	// helps prevent the network from becoming another public test network
	// since it will not be able to learn about other peers that have not
	// specifically been provided.
	if cfg.SimNet {
		return
	}

	// Ignore old style addresses which don't include a timestamp.
	if p.ProtocolVersion() < wire.NetAddressTimeVersion {
		return
	}

	// A message that has no addresses is invalid.
	if len(msg.AddrList) == 0 {
		p.logError("Command [%s] from %s does not contain any addresses",
			msg.Command(), p)
		p.Disconnect()
		return
	}

	for _, na := range msg.AddrList {
		// Don't add more address if we're disconnecting.
		if atomic.LoadInt32(&p.disconnect) != 0 {
			return
		}

		// Set the timestamp to 5 days ago if it's more than 24 hours
		// in the future so this address is one of the first to be
		// removed when space is needed.
		now := time.Now()
		if na.Timestamp.After(now.Add(time.Minute * 10)) {
			na.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}

		// Add address to known addresses for this peer.
		p.knownAddresses[addrmgr.NetAddressKey(na)] = struct{}{}
	}

	// Add addresses to server address manager.  The address manager handles
	// the details of things such as preventing duplicate addresses, max
	// addresses, and last seen updates.
	// XXX factomd gives a 2 hour time penalty here, do we want to do the
	// same?
	p.server.addrManager.AddAddresses(msg.AddrList, p.na)
}

// handlePingMsg is invoked when a peer receives a ping factom message.  For
// recent clients (protocol version > BIP0031Version), it replies with a pong
// message.  For older clients, it does nothing and anything other than failure
// is considered a successful ping.
func (p *peer) handlePingMsg(msg *wire.MsgPing) {
	// Only Reply with pong is message comes from a new enough client.
	if p.ProtocolVersion() > wire.BIP0031Version {
		// Include nonce from ping so pong can be identified.
		p.QueueMessage(wire.NewMsgPong(msg.Nonce), nil)
	}
}

// handlePongMsg is invoked when a peer received a pong factom message.
// recent clients (protocol version > BIP0031Version), and if we had send a ping
// previosuly we update our ping time statistics. If the client is too old or
// we had not send a ping we ignore it.
func (p *peer) handlePongMsg(msg *wire.MsgPong) {
	p.StatsMtx.Lock()
	defer p.StatsMtx.Unlock()

	// Arguably we could use a buffered channel here sending data
	// in a fifo manner whenever we send a ping, or a list keeping track of
	// the times of each ping. For now we just make a best effort and
	// only record stats if it was for the last ping sent. Any preceding
	// and overlapping pings will be ignored. It is unlikely to occur
	// without large usage of the ping rpc call since we ping
	// infrequently enough that if they overlap we would have timed out
	// the peer.
	if p.protocolVersion > wire.BIP0031Version &&
		p.lastPingNonce != 0 && msg.Nonce == p.lastPingNonce {
		p.lastPingMicros = time.Now().Sub(p.lastPingTime).Nanoseconds()
		p.lastPingMicros /= 1000 // convert to usec.
		p.lastPingNonce = 0
	}
}

// readMessage reads the next factom message from the peer with logging.
func (p *peer) readMessage() (wire.Message, []byte, error) {
	n, msg, buf, err := wire.ReadMessageN(p.conn, p.ProtocolVersion(),
		p.fctnet)
	p.StatsMtx.Lock()
	p.bytesReceived += uint64(n)
	p.StatsMtx.Unlock()
	p.server.AddBytesReceived(uint64(n))
	if err != nil {
		return nil, nil, err
	}

	// Use closures to log expensive operations so they are only run when
	// the logging level requires it.
	/*peerLog.Debugf("read: %v", newLogClosure(func() string {
		// Debug summary of message.
		summary := messageSummary(msg)
		if len(summary) > 0 {
			summary = " (" + summary + ")"
		}
		return fmt.Sprintf("Received %v%s from %s",
			msg.Command(), summary, p)
	}))
	peerLog.Debugf("read: %v", newLogClosure(func() string {
		return spew.Sdump(msg)
	}))
		peerLog.Debugf("%v", newLogClosure(func() string {
		return spew.Sdump(buf)
	})) */

	return msg, buf, nil
}

// writeMessage sends a factom Message to the peer with logging.
func (p *peer) writeMessage(msg wire.Message) {
	// Don't do anything if we're disconnecting.
	if atomic.LoadInt32(&p.disconnect) != 0 {
		return
	}
	if !p.VersionKnown() {
		switch msg.(type) {
		case *wire.MsgVersion:
			// This is OK.
		case *wire.MsgReject:
			// This is OK.
		default:
			// Drop all messages other than version and reject if
			// the handshake has not already been done.
			return
		}
	}

	// Use closures to log expensive operations so they are only run when
	// the logging level requires it.
	/*peerLog.Debugf("write: %v", newLogClosure(func() string {
		// Debug summary of message.
		summary := messageSummary(msg)
		if len(summary) > 0 {
			summary = " (" + summary + ")"
		}
		return fmt.Sprintf("Sending %v%s to %s", msg.Command(),
			summary, p)
	}))
	peerLog.Debugf("write: %v", newLogClosure(func() string {
		return spew.Sdump(msg)
	}))
		peerLog.Debugf("%v", newLogClosure(func() string {
		var buf bytes.Buffer
		err := wire.WriteMessage(&buf, msg, p.ProtocolVersion(),
			p.fctnet)
		if err != nil {
			return err.Error()
		}
		return spew.Sdump(buf.Bytes())
	})) */

	// Write the message to the peer.
	n, err := wire.WriteMessageN(p.conn, msg, p.ProtocolVersion(),
		p.fctnet)
	p.StatsMtx.Lock()
	p.bytesSent += uint64(n)
	p.StatsMtx.Unlock()
	p.server.AddBytesSent(uint64(n))
	if err != nil {
		p.Disconnect()
		p.logError("Can't send message to %s: %v", p, err)
		return
	}
}

// isAllowedByRegression returns whether or not the passed error is allowed by
// regression tests without disconnecting the peer.  In particular, regression
// tests need to be allowed to send malformed messages without the peer being
// disconnected.
func (p *peer) isAllowedByRegression(err error) bool {
	// Don't allow the error if it's not specifically a malformed message
	// error.
	if _, ok := err.(*wire.MessageError); !ok {
		return false
	}

	// Don't allow the error if it's not coming from localhost or the
	// hostname can't be determined for some reason.
	host, _, err := net.SplitHostPort(p.addr)
	if err != nil {
		return false
	}

	if host != "127.0.0.1" && host != "localhost" {
		return false
	}

	// Allowed if all checks passed.
	return true
}

// inHandler handles all incoming messages for the peer.  It must be run as a
// goroutine.
func (p *peer) inHandler() {
	// Peers must complete the initial version negotiation within a shorter
	// timeframe than a general idle timeout.  The timer is then reset below
	// to idleTimeoutMinutes for all future messages.
	idleTimer := time.AfterFunc(negotiateTimeoutSeconds*time.Second, func() {
		if p.VersionKnown() {
			peerLog.Warnf("Peer %s no answer for %d minutes, "+
				"disconnecting", p, idleTimeoutMinutes)
		}
		p.Disconnect()
	})
out:
	for atomic.LoadInt32(&p.disconnect) == 0 {
		rmsg, buf, err := p.readMessage()
		// Stop the timer now, if we go around again we will reset it.
		idleTimer.Stop()
		if err != nil {
			// In order to allow regression tests with malformed
			// messages, don't disconnect the peer when we're in
			// regression test mode and the error is one of the
			// allowed errors.
			if cfg.RegressionTest && p.isAllowedByRegression(err) {
				peerLog.Errorf("Allowed regression test "+
					"error from %s: %v", p, err)
				idleTimer.Reset(idleTimeoutMinutes * time.Minute)
				continue
			}

			// Only log the error and possibly send reject message
			// if we're not forcibly disconnecting.
			if atomic.LoadInt32(&p.disconnect) == 0 {
				errMsg := fmt.Sprintf("Can't read message "+
					"from %s: %v", p, err)
				p.logError(errMsg)

				// Only send the reject message if it's not
				// because the remote client disconnected.
				if err != io.EOF {
					// Push a reject message for the
					// malformed message and wait for the
					// message to be sent before
					// disconnecting.
					//
					// NOTE: Ideally this would include the
					// command in the header if at least
					// that much of the message was valid,
					// but that is not currently exposed by
					// wire, so just used malformed for the
					// command.
					p.PushRejectMsg("malformed",
						wire.RejectMalformed, errMsg,
						nil, true)
				}

			}
			break out
		}
		p.StatsMtx.Lock()
		p.lastRecv = time.Now()
		p.StatsMtx.Unlock()

		// Ensure version message comes first.
		if vmsg, ok := rmsg.(*wire.MsgVersion); !ok && !p.VersionKnown() {
			errStr := "A version message must precede all others"
			p.logError(errStr)

			// Push a reject message and wait for the message to be
			// sent before disconnecting.
			p.PushRejectMsg(vmsg.Command(), wire.RejectMalformed,
				errStr, nil, true)
			break out
		}

		// Handle each supported message type.
		switch msg := rmsg.(type) {
		case *wire.MsgVersion:
			p.handleVersionMsg(msg)

		case *wire.MsgVerAck:
			p.StatsMtx.Lock()
			versionSent := p.versionSent
			verAckReceived := p.verAckReceived
			p.StatsMtx.Unlock()

			if !versionSent {
				peerLog.Infof("Received 'verack' from peer %v "+
					"before version was sent -- disconnecting", p)
				break out
			}
			if verAckReceived {
				peerLog.Infof("Already received 'verack' from "+
					"peer %v -- disconnecting", p)
				break out
			}
			p.verAckReceived = true

		case *wire.MsgGetAddr:
			p.handleGetAddrMsg(msg)

		case *wire.MsgAddr:
			p.handleAddrMsg(msg)

		case *wire.MsgPing:
			p.handlePingMsg(msg)

		case *wire.MsgPong:
			p.handlePongMsg(msg)

		case *wire.MsgAlert:
			// Intentionally ignore alert messages.
			//
			// The reference client currently bans peers that send
			// alerts not signed with its key.  We could verify
			// against their key, but since the reference client
			// is currently unwilling to support other
			// implementions' alert messages, we will not relay
			// theirs.

		case *wire.MsgNotFound:
			// TODO(davec): Ignore this for now, but ultimately
			// it should probably be used to detect when something
			// we requested needs to be re-requested from another
			// peer.

		case *wire.MsgReject:
			// Logging of the rejected message is handled already in readMessage.
			if msg.Code == wire.RejectInvalidPWd || msg.Code == wire.RejectInvalidSig {
				// for testing purpose
				panic(msg.Reason)
			}


			// Factom transactional messages processed only by servers
			// but they will be relayed through OutMsgQueue by clients
		case *wire.MsgCommitChain:
			p.handleCommitChainMsg(msg)
			//p.FactomRelay(msg)

		case *wire.MsgCommitEntry:
			p.handleCommitEntryMsg(msg)
			//p.FactomRelay(msg)

		case *wire.MsgRevealEntry:
			p.handleRevealEntryMsg(msg)
			//p.FactomRelay(msg)

		case *wire.MsgFactoidTX:
			p.handleFactoidMsg(msg, buf)
			//p.FactomRelay(msg)


			// Factom Control messages processed only by servers
			// no relay or reach by clients
		case *wire.MsgAck:
			p.handleAckMsg(msg)

		case *wire.MsgDirBlockSig:
			p.handleDirBlockSigMsg(msg)

		case *wire.MsgCandidate:
			p.handleCandidateMsg(msg)

		case *wire.MsgNextLeader:
			p.handleNextLeaderMsg(msg)

		case *wire.MsgNextLeaderResp:
			p.handleNextLeaderRespMsg(msg)

		case *wire.MsgMissing:
			p.handleMissingMsg(msg)

		case *wire.MsgCurrentLeader:
			p.handleCurrentLeaderMsg(msg)


			// Factom blocks downloading
		case *wire.MsgGetDirBlocks:
			p.handleGetDirBlocksMsg(msg)

		case *wire.MsgDirInv:
			p.handleDirInvMsg(msg)

		case *wire.MsgGetDirData:
			p.handleGetDirDataMsg(msg)

		case *wire.MsgDirBlock:
			p.handleDirBlockMsg(msg, buf)

		case *wire.MsgGetNonDirData:
			p.handleGetNonDirDataMsg(msg)

		case *wire.MsgABlock:
			p.handleABlockMsg(msg, buf)

		case *wire.MsgECBlock:
			p.handleECBlockMsg(msg, buf)

		case *wire.MsgEBlock:
			p.handleEBlockMsg(msg, buf)

		case *wire.MsgFBlock:
			p.handleFBlockMsg(msg, buf)

		case *wire.MsgGetEntryData:
			p.handleGetEntryDataMsg(msg)

		case *wire.MsgEntry:
			p.handleEntryMsg(msg, buf)

			// download individual block by hash/height
		case *wire.MsgGetFactomData:
			p.handleGetFactomDataMsg(msg)
			
		default:
			peerLog.Debugf("Received unhandled message of type %v: Fix Me",
				rmsg.Command())
		}

		// ok we got a message, reset the timer.
		// timer just calls p.Disconnect() after logging.
		idleTimer.Reset(idleTimeoutMinutes * time.Minute)
		p.retryCount = 0
	}

	idleTimer.Stop()

	// Ensure connection is closed and notify the server that the peer is
	// done.
	p.Disconnect()
	p.server.donePeers <- p

	// Only tell block manager we are gone if we ever told it we existed.
	if p.VersionKnown() {
		p.server.blockManager.DonePeer(p)
	}

	peerLog.Tracef("Peer input handler done for %s", p)
}

// queueHandler handles the queueing of outgoing data for the peer. This runs
// as a muxer for various sources of input so we can ensure that blockmanager
// and the server goroutine both will not block on us sending a message.
// We then pass the data on to outHandler to be actually written.
func (p *peer) queueHandler() {
	p.queueWg.Add(1)
	defer func() {
		//fmt.Println("queueWg.Done for queueHandler")
		p.queueWg.Done()
	}()

	pendingMsgs := list.New()
	invSendQueue := list.New()
	trickleTicker := time.NewTicker(time.Second * 10)
	defer trickleTicker.Stop()

	// We keep the waiting flag so that we know if we have a message queued
	// to the outHandler or not.  We could use the presence of a head of
	// the list for this but then we have rather racy concerns about whether
	// it has gotten it at cleanup time - and thus who sends on the
	// message's done channel.  To avoid such confusion we keep a different
	// flag and pendingMsgs only contains messages that we have not yet
	// passed to outHandler.
	waiting := false

	// To avoid duplication below.
	queuePacket := func(msg outMsg, list *list.List, waiting bool) bool {
		if !waiting {
			// peerLog.Tracef("%s: sending to outHandler", p)
			p.sendQueue <- msg
			// peerLog.Tracef("%s: sent to outHandler", p)
		} else {
			list.PushBack(msg)
		}
		// we are always waiting now.
		return true
	}
out:
	for {
		select {
		case msg := <-p.outputQueue:
			waiting = queuePacket(msg, pendingMsgs, waiting)

		// This channel is notified when a message has been sent across
		// the network socket.
		case <-p.sendDoneQueue:
			// peerLog.Tracef("%s: acked by outhandler", p)

			// No longer waiting if there are no more messages
			// in the pending messages queue.
			next := pendingMsgs.Front()
			if next == nil {
				waiting = false
				continue
			}

			// Notify the outHandler about the next item to
			// asynchronously send.
			val := pendingMsgs.Remove(next)
			// peerLog.Tracef("%s: sending to outHandler", p)
			p.sendQueue <- val.(outMsg)
			// peerLog.Tracef("%s: sent to outHandler", p)

		case iv := <-p.outputInvChan:
			// No handshake?  They'll find out soon enough.
			if p.VersionKnown() {
				invSendQueue.PushBack(iv)
			}

		case <-trickleTicker.C:
			// Don't send anything if we're disconnecting or there
			// is no queued inventory.
			// version is known if send queue has any entries.
			if atomic.LoadInt32(&p.disconnect) != 0 ||
				invSendQueue.Len() == 0 {
				continue
			}

			// Create and send as many inv messages as needed to
			// drain the inventory send queue.
			//invMsg := wire.NewMsgInv()
			invMsg := wire.NewMsgDirInv()
			for e := invSendQueue.Front(); e != nil; e = invSendQueue.Front() {
				iv := invSendQueue.Remove(e).(*wire.InvVect)

				// Don't send inventory that became known after
				// the initial check.
				if p.isKnownInventory(iv) {
					continue
				}

				invMsg.AddInvVect(iv)
				if len(invMsg.InvList) >= maxInvTrickleSize {
					waiting = queuePacket(
						outMsg{msg: invMsg},
						pendingMsgs, waiting)
					//invMsg = wire.NewMsgInv()
					invMsg = wire.NewMsgDirInv()
				}

				// Add the inventory that is being relayed to
				// the known inventory for the peer.
				p.AddKnownInventory(iv)
			}
			if len(invMsg.InvList) > 0 {
				waiting = queuePacket(outMsg{msg: invMsg},
					pendingMsgs, waiting)
			}

		case <-p.quit:
			break out
		}
	}

	// Drain any wait channels before we go away so we don't leave something
	// waiting for us.
	for e := pendingMsgs.Front(); e != nil; e = pendingMsgs.Front() {
		val := pendingMsgs.Remove(e)
		msg := val.(outMsg)
		if msg.doneChan != nil {
			msg.doneChan <- struct{}{}
		}
	}
cleanup:
	for {
		select {
		case msg := <-p.outputQueue:
			if msg.doneChan != nil {
				msg.doneChan <- struct{}{}
			}
		case <-p.outputInvChan:
			// Just drain channel
		// sendDoneQueue is buffered so doesn't need draining.
		default:
			break cleanup
		}
	}
	// peerLog.Tracef("Peer queue handler done for %s", p)
}

// outHandler handles all outgoing messages for the peer.  It must be run as a
// goroutine.  It uses a buffered channel to serialize output messages while
// allowing the sender to continue running asynchronously.
func (p *peer) outHandler() {
	pingTimer := time.AfterFunc(pingTimeoutMinutes*time.Minute, func() {
		nonce, err := wire.RandomUint64()
		if err != nil {
			peerLog.Errorf("Not sending ping on timeout to %s: %v",
				p, err)
			return
		}
		p.QueueMessage(wire.NewMsgPing(nonce), nil)
	})
out:
	for {
		select {
		case msg := <-p.sendQueue:
			// If the message is one we should get a reply for
			// then reset the timer, we only want to send pings
			// when otherwise we would not receive a reply from
			// the peer. We specifically do not count block or inv
			// messages here since they are not sure of a reply if
			// the inv is of no interest explicitly solicited invs
			// should elicit a reply but we don't track them
			// specially.
			// peerLog.Tracef("%s: received from queuehandler", p)
			reset := true
			switch m := msg.msg.(type) {
			case *wire.MsgVersion:
				// should get a verack
				p.StatsMtx.Lock()
				p.versionSent = true
				p.StatsMtx.Unlock()
			case *wire.MsgGetAddr:
				// should get addresses
			case *wire.MsgPing:
				// expects pong
				// Also set up statistics.
				p.StatsMtx.Lock()
				//if p.protocolVersion > wire.BIP0031Version {
				p.lastPingNonce = m.Nonce
				p.lastPingTime = time.Now()
				//}
				p.StatsMtx.Unlock()
			case *wire.MsgGetDirData:
				// Should get us dir block, or not found for factom
			default:
				// Not one of the above, no sure reply.
				// We want to ping if nothing else
				// interesting happens.
				reset = false
			}
			if reset {
				pingTimer.Reset(pingTimeoutMinutes * time.Minute)
			}
			p.writeMessage(msg.msg)
			p.StatsMtx.Lock()
			p.lastSend = time.Now()
			p.StatsMtx.Unlock()
			if msg.doneChan != nil {
				msg.doneChan <- struct{}{}
			}
			// peerLog.Tracef("%s: acking queuehandler", p)
			p.sendDoneQueue <- struct{}{}
			// peerLog.Tracef("%s: acked queuehandler", p)

		case <-p.quit:
			break out
		}
	}

	pingTimer.Stop()

	p.queueWg.Wait()

	// Drain any wait channels before we go away so we don't leave something
	// waiting for us. We have waited on queueWg and thus we can be sure
	// that we will not miss anything sent on sendQueue.
cleanup:
	for {
		select {
		case msg := <-p.sendQueue:
			if msg.doneChan != nil {
				msg.doneChan <- struct{}{}
			}
			// no need to send on sendDoneQueue since queueHandler
			// has been waited on and already exited.
		default:
			break cleanup
		}
	}
	// peerLog.Tracef("Peer output handler done for %s", p)
}

// QueueMessage adds the passed factom message to the peer send queue.  It
// uses a buffered channel to communicate with the output handler goroutine so
// it is automatically rate limited and safe for concurrent access.
func (p *peer) QueueMessage(msg wire.Message, doneChan chan struct{}) {
	// Avoid risk of deadlock if goroutine already exited. The goroutine
	// we will be sending to hangs around until it knows for a fact that
	// it is marked as disconnected. *then* it drains the channels.
	if !p.Connected() {
		// avoid deadlock...
		if doneChan != nil {
			go func() {
				doneChan <- struct{}{}
			}()
		}
		return
	}
	p.outputQueue <- outMsg{msg: msg, doneChan: doneChan}
}

// QueueInventory adds the passed inventory to the inventory send queue which
// might not be sent right away, rather it is trickled to the peer in batches.
// Inventory that the peer is already known to have is ignored.  It is safe for
// concurrent access.
func (p *peer) QueueInventory(invVect *wire.InvVect) {
	// Don't add the inventory to the send queue if the peer is
	// already known to have it.
	if p.isKnownInventory(invVect) {
		return
	}

	// Avoid risk of deadlock if goroutine already exited. The goroutine
	// we will be sending to hangs around until it knows for a fact that
	// it is marked as disconnected. *then* it drains the channels.
	if !p.Connected() {
		return
	}

	p.outputInvChan <- invVect
}

// Connected returns whether or not the peer is currently connected.
func (p *peer) Connected() bool {
	return atomic.LoadInt32(&p.connected) != 0 &&
		atomic.LoadInt32(&p.disconnect) == 0
}

// Disconnect disconnects the peer by closing the connection.  It also sets
// a flag so the impending shutdown can be detected.
func (p *peer) Disconnect() {
	// did we win the race?
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}

	// Update the address' last seen time if the peer has acknowledged
	// our version and has sent us its version as well.
	p.StatsMtx.Lock()
	if p.verAckReceived && p.versionKnown && p.na != nil {
		p.server.addrManager.Connected(p.na)
	}
	p.StatsMtx.Unlock()

	peerLog.Tracef("disconnecting %s", p)
	close(p.quit)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.conn.Close()
	}
}

// Start begins processing input and output messages.  It also sends the initial
// version message for outbound connections to start the negotiation process.
func (p *peer) Start() error {
	// Already started?
	if atomic.AddInt32(&p.started, 1) != 1 {
		return nil
	}

	peerLog.Tracef("Starting peer %s", p)

	// Send an initial version message if this is an outbound connection.
	if !p.inbound {
		peerLog.Infof("peer.Start(): !p.inbound, before pushVersionMsg: %s", p)
		err := p.pushVersionMsg()
		if err != nil {
			p.logError("Can't send outbound version message %v", err)
			p.Disconnect()
			return err
		}
	}

	// Start processing input and output.
	go p.inHandler()
	// queueWg is kept so that outHandler knows when the queue has exited so
	// it can drain correctly.
	go p.queueHandler()
	go p.outHandler()

	return nil
}

// Shutdown gracefully shuts down the peer by disconnecting it.
func (p *peer) Shutdown() {
	peerLog.Tracef("Shutdown peer %s", p)
	p.Disconnect()
}

// newPeerBase returns a new base factom peer for the provided server and
// inbound flag.  This is used by the newInboundPeer and newOutboundPeer
// functions to perform base setup needed by both types of peers.
func newPeerBase(s *server, inbound bool) *peer {
	p := peer{
		server:          s,
		protocolVersion: maxProtocolVersion,
		fctnet:          s.chainParams.Net,
		services:        wire.SFNodeNetwork,
		inbound:         inbound,
		knownAddresses:  make(map[string]struct{}),
		knownInventory:  NewMruInventoryMap(maxKnownInventory),
		// requestedTxns:   make(map[wire.ShaHash]struct{}),
		// requestedBlocks: make(map[wire.ShaHash]struct{}),
		outputQueue:     make(chan outMsg, outputBufferSize),
		sendQueue:       make(chan outMsg, 1),   // nonblocking sync
		sendDoneQueue:   make(chan struct{}, 1), // nonblocking sync
		outputInvChan:   make(chan *wire.InvVect, outputBufferSize),
		txProcessed:     make(chan struct{}, 1),
		blockProcessed:  make(chan struct{}, 1),
		quit:            make(chan struct{}),
	}
	return &p
}

// newInboundPeer returns a new inbound factom peer for the provided server and
// connection.  Use Start to begin processing incoming and outgoing messages.
func newInboundPeer(s *server, conn net.Conn) *peer {
	p := newPeerBase(s, true)
	p.conn = conn
	p.addr = conn.RemoteAddr().String()
	p.timeConnected = time.Now()
	atomic.AddInt32(&p.connected, 1)
	return p
}

// newOutbountPeer returns a new outbound factom peer for the provided server and
// address and connects to it asynchronously. If the connection is successful
// then the peer will also be started.
func newOutboundPeer(s *server, addr string, persistent bool, retryCount int64) *peer {
	p := newPeerBase(s, false)
	p.addr = addr
	p.persistent = persistent
	p.retryCount = retryCount

	// Setup p.na with a temporary address that we are connecting to with
	// faked up service flags.  We will replace this with the real one after
	// version negotiation is successful.  The only failure case here would
	// be if the string was incomplete for connection so can't be split
	// into address and port, and thus this would be invalid anyway.  In
	// which case we return nil to be handled by the caller.  This must be
	// done before we fork off the goroutine because as soon as this
	// function returns the peer must have a valid netaddress.
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		p.logError("Tried to create a new outbound peer with invalid "+
			"address %s: %v", addr, err)
		return nil
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		p.logError("Tried to create a new outbound peer with invalid "+
			"port %s: %v", portStr, err)
		return nil
	}

	p.na, err = s.addrManager.HostToNetAddress(host, uint16(port), 0)
	if err != nil {
		p.logError("Can not turn host %s into netaddress: %v",
			host, err)
		return nil
	}

	go func() {
		if atomic.LoadInt32(&p.disconnect) != 0 {
			return
		}
		if p.retryCount > 0 {
			scaledInterval := connectionRetryInterval.Nanoseconds() * p.retryCount / 2
			scaledDuration := time.Duration(scaledInterval)
			if scaledDuration > maxConnectionRetryInterval {
				scaledDuration = maxConnectionRetryInterval
			}
			srvrLog.Debugf("Retrying connection to %s in %s", addr, scaledDuration)
			time.Sleep(scaledDuration)
		}
		srvrLog.Debugf("Attempting to connect to %s", addr)
		conn, err := factomdDial("tcp", addr)
		if err != nil {
			srvrLog.Debugf("Failed to connect to %s: %v", addr, err)
			p.server.donePeers <- p
			return
		}

		// We may have slept and the server may have scheduled a shutdown.  In that
		// case ditch the peer immediately.
		if atomic.LoadInt32(&p.disconnect) == 0 {
			p.timeConnected = time.Now()
			p.server.addrManager.Attempt(p.na)

			// Connection was successful so log it and start peer.
			srvrLog.Debugf("Connected to %s", conn.RemoteAddr())
			p.conn = conn
			atomic.AddInt32(&p.connected, 1)
			p.Start()
		}
	}()
	return p
}

// logError makes sure that we only log errors loudly on user peers.
func (p *peer) logError(fmt string, args ...interface{}) {
	if p.persistent {
		peerLog.Errorf(fmt, args...)
	} else {
		peerLog.Debugf(fmt, args...)
	}
}

func isVersionMismatch(us, them int32) bool {
	peerLog.Debug(fmt.Sprintf("VERSION: us=%d , peer= %d", us, them))

	if us != them {
		peerLog.Debug("NEED CLIENT UPGRADE !!!")
		return true
	}

	return false
}

// handleFBlockMsg is invoked when a peer receives a factoid block message.
func (p *peer) handleFBlockMsg(msg *wire.MsgFBlock, buf []byte) {
	processFBlock(msg)
	binary, _ := msg.SC.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())

	iv := wire.NewInvVect(wire.InvTypeFactomFBlock, hash)
	p.AddKnownInventory(iv)
	//inMsgQueue <- msg
}

// handleDirBlockMsg is invoked when a peer receives a dir block message.
func (p *peer) handleDirBlockMsg(msg *wire.MsgDirBlock, buf []byte) {
	//fmt.Println("handleDirBlockMsg: dir block msg: ", spew.Sdump(msg.DBlk.Header))
	processDirBlock(msg)
	//inMsgQueue <- msg
	//fmt.Println("handleDirBlockMsg: inMsgQ.len: ", len(inMsgQueue))

	binary, _ := msg.DBlk.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())

	iv := wire.NewInvVect(wire.InvTypeFactomDirBlock, hash)
	p.AddKnownInventory(iv)

	if msg.NonDirBlockNeeded {
		p.pushGetNonDirDataMsg(msg.DBlk)
	}

	// delete(p.requestedBlocks, *hash)
	// delete(p.server.blockManager.requestedBlocks, *hash)

	// Meta-data about the new block this peer is reporting. We use this
	// below to update this peer's lastest block height and the heights of
	// other peers based on their last announced block sha. This allows us
	// to dynamically update the block heights of peers, avoiding stale heights
	// when looking for a new sync peer. Upon acceptance of a block or
	// recognition of an orphan, we also use this information to update
	// the block heights over other peers who's invs may have been ignored
	// if we are actively syncing while the chain is not yet current or
	// who may have lost the lock announcment race.
	var heightUpdate int32

	// Query the db for the latest best block since the block
	// that was processed could be on a side chain or have caused
	// a reorg.
	blkShaUpdate, newestHeight, _ := db.FetchBlockHeightCache()

	// Update this peer's latest block height, for future
	// potential sync node candidancy.
	heightUpdate = int32(newestHeight)

	// Update the block height for this peer. But only send a message to
	// the server for updating peer heights if this is an orphan or our
	// chain is "current". This avoid sending a spammy amount of messages
	// if we're syncing the chain from scratch.
	if blkShaUpdate != nil && heightUpdate != 0 {
		p.UpdateLastBlockHeight(heightUpdate)
		peerLog.Infof("handleDirBlockMsg: UpdateLastBlockHeight: %d, %s, %s",
			heightUpdate, blkShaUpdate, p)
		if p.server.blockManager.current() {
			peerLog.Infof("handleDirBlockMsg: UpdatePeerHeights: %d, %s, %s",
				heightUpdate, blkShaUpdate, p)
			go p.server.UpdatePeerHeights(blkShaUpdate, int32(heightUpdate), p)
		}
	}
}

// handleABlockMsg is invoked when a peer receives a entry credit block message.
func (p *peer) handleABlockMsg(msg *wire.MsgABlock, buf []byte) {
	processABlock(msg)
	binary, _ := msg.ABlk.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())

	iv := wire.NewInvVect(wire.InvTypeFactomAdminBlock, hash)
	p.AddKnownInventory(iv)
	//inMsgQueue <- msg
}

// handleECBlockMsg is invoked when a peer receives a entry credit block
// message.
func (p *peer) handleECBlockMsg(msg *wire.MsgECBlock, buf []byte) {
	procesECBlock(msg)
	headerHash, err := msg.ECBlock.HeaderHash()
	if err != nil {
		panic(err)
	}
	hash := wire.FactomHashToShaHash(headerHash)
	iv := wire.NewInvVect(wire.InvTypeFactomEntryCreditBlock, hash)
	p.AddKnownInventory(iv)
	//inMsgQueue <- msg
}

// handleEBlockMsg is invoked when a peer receives an entry block factom message.
func (p *peer) handleEBlockMsg(msg *wire.MsgEBlock, buf []byte) {
	processEBlock(msg)
	binary, _ := msg.EBlk.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())
	iv := wire.NewInvVect(wire.InvTypeFactomEntryBlock, hash)
	p.AddKnownInventory(iv)
	if msg.EntryNeeded {
		p.pushGetEntryDataMsg(msg.EBlk)
	}
	//inMsgQueue <- msg
}

// handleEntryMsg is invoked when a peer receives a EBlock Entry message.
func (p *peer) handleEntryMsg(msg *wire.MsgEntry, buf []byte) {
	processEntry(msg)
	binary, _ := msg.Entry.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())
	iv := wire.NewInvVect(wire.InvTypeFactomEntry, hash)
	p.AddKnownInventory(iv)
	//inMsgQueue <- msg
}

// handleGetEntryDataMsg is invoked when a peer receives a get entry data message and
// is used to deliver entry of EBlock information.
func (p *peer) handleGetEntryDataMsg(msg *wire.MsgGetEntryData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()

	// We wait on the this wait channel periodically to prevent queueing
	// far more data than we can send in a reasonable time, wasting memory.
	// The waiting occurs after the database fetch for the next one to
	// provide a little pipelining.

	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)
	for i, iv := range msg.InvList {

		var c chan struct{}
		// If this will be the last message we send.
		if i == len(msg.InvList)-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else { //if (i+1)%3 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}

		if iv.Type != wire.InvTypeFactomEntry {
			continue
		}

		// Is this right? what is iv.hash?
		blk, err := db.FetchEBlockByHash(iv.Hash.ToFactomHash())

		if err != nil {

			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return
		}

		for _, ebEntry := range blk.Body.EBEntries {

			//Skip the minute markers
			if ebEntry.IsMinuteMarker() {
				continue
			}
			var err error
			err = p.pushEntryMsg(ebEntry, c, waitChan)
			if err != nil {
				notFound.AddInvVect(iv)
				// When there is a failure fetching the final entry
				// and the done channel was sent in due to there
				// being no outstanding not found inventory, consume
				// it here because there is now not found inventory
				// that will use the channel momentarily.
				if i == len(msg.InvList)-1 && c != nil {
					<-c
				}
			}
			numAdded++
			waitChan = c
		}

	}
	if len(notFound.InvList) != 0 {
		p.QueueMessage(notFound, doneChan)
	}

	// Wait for messages to be sent. We can send quite a lot of data at this
	// point and this will keep the peer busy for a decent amount of time.
	// We don't process anything else by them in this time so that we
	// have an idea of when we should hear back from them - else the idle
	// timeout could fire when we were only half done sending the blocks.
	if numAdded > 0 {
		<-doneChan
	}
}

// handleGetNonDirDataMsg is invoked when a peer receives a dir block message.
// It returns the corresponding data block like Factoid block,
// EC block, Entry block, and Entry based on directory block's ChainID
func (p *peer) handleGetNonDirDataMsg(msg *wire.MsgGetNonDirData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()

	// We wait on the this wait channel periodically to prevent queueing
	// far more data than we can send in a reasonable time, wasting memory.
	// The waiting occurs after the database fetch for the next one to
	// provide a little pipelining.

	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)
	for i, iv := range msg.InvList {
		var c chan struct{}
		// If this will be the last message we send.
		if i == len(msg.InvList)-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else { //if (i+1)%3 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}

		if iv.Type != wire.InvTypeFactomNonDirBlock {
			continue
		}

		// Is this right? what is iv.hash?
		blk, err := db.FetchDBlockByHash(iv.Hash.ToFactomHash())

		if err != nil {
			// check newly generated dir block in processor, 
			var hash *common.Hash
			if newDBlock != nil {
				bin, _ := newDBlock.MarshalBinary()
				hash = common.Sha(bin)
			}
			if hash != nil && bytes.Compare(iv.Hash.ToFactomHash().Bytes(), hash.Bytes()) == 0 {
				fmt.Println("found dirBlock as the newly generated dir block: ", 
					newDBlock.Header.DBHeight)
				blk = newDBlock
			} else {
				peerLog.Tracef("Unable to fetch requested dir block sha %v: %v",
					iv.Hash, err)

				if doneChan != nil {
					doneChan <- struct{}{}
				}
				return 
			}
		}

		for _, dbEntry := range blk.DBEntries {

			var err error
			switch dbEntry.ChainID.String() {
			case hex.EncodeToString(common.EC_CHAINID[:]):
				err = p.pushECBlockMsg(dbEntry.KeyMR, c, waitChan)

			case hex.EncodeToString(common.ADMIN_CHAINID[:]):
				err = p.pushABlockMsg(dbEntry.KeyMR, c, waitChan)

			case hex.EncodeToString(common.FACTOID_CHAINID[:]): //wire.FChainID.String():
				err = p.pushFBlockMsg(dbEntry.KeyMR, c, waitChan)

			default:
				err = p.pushEBlockMsg(dbEntry.KeyMR, c, waitChan)
				//continue
			}
			if err != nil {
				notFound.AddInvVect(iv)
				// When there is a failure fetching the final entry
				// and the done channel was sent in due to there
				// being no outstanding not found inventory, consume
				// it here because there is now not found inventory
				// that will use the channel momentarily.
				if i == len(msg.InvList)-1 && c != nil {
					<-c
				}
			}
			numAdded++
			waitChan = c
		}

	}
	if len(notFound.InvList) != 0 {
		p.QueueMessage(notFound, doneChan)
	}

	// Wait for messages to be sent. We can send quite a lot of data at this
	// point and this will keep the peer busy for a decent amount of time.
	// We don't process anything else by them in this time so that we
	// have an idea of when we should hear back from them - else the idle
	// timeout could fire when we were only half done sending the blocks.
	if numAdded > 0 {
		<-doneChan
	}
}

// handleDirInvMsg is invoked when a peer receives an inv factom message and is
// used to examine the inventory being advertised by the remote peer and react
// accordingly.  We pass the message down to blockmanager which will call
// QueueMessage with any appropriate responses.
func (p *peer) handleDirInvMsg(msg *wire.MsgDirInv) {
	p.server.blockManager.QueueDirInv(msg, p)
}

// handleGetDirDataMsg is invoked when a peer receives a getdirdata message and
// is used to deliver block and transaction information.
func (p *peer) handleGetDirDataMsg(msg *wire.MsgGetDirData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()

	// We wait on the this wait channel periodically to prevent queueing
	// far more data than we can send in a reasonable time, wasting memory.
	// The waiting occurs after the database fetch for the next one to
	// provide a little pipelining.
	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)

	for i, iv := range msg.InvList {
		var c chan struct{}
		// If this will be the last message we send.
		if i == len(msg.InvList)-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else if (i+1)%3 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}
		var err error
		switch iv.Type {
		case wire.InvTypeFactomDirBlock:
			err = p.pushDirBlockMsg(&iv.Hash, c, waitChan)
			/*
				case wire.InvTypeFilteredBlock:
					err = p.pushMerkleBlockMsg(&iv.Hash, c, waitChan)
			*/
		default:
			peerLog.Warnf("Unknown type in inventory request %d",
				iv.Type)
			continue
		}
		if err != nil {
			notFound.AddInvVect(iv)

			// When there is a failure fetching the final entry
			// and the done channel was sent in due to there
			// being no outstanding not found inventory, consume
			// it here because there is now not found inventory
			// that will use the channel momentarily.
			if i == len(msg.InvList)-1 && c != nil {
				<-c
			}
		}
		numAdded++
		waitChan = c
	}
	if len(notFound.InvList) != 0 {
		p.QueueMessage(notFound, doneChan)
	}

	// Wait for messages to be sent. We can send quite a lot of data at this
	// point and this will keep the peer busy for a decent amount of time.
	// We don't process anything else by them in this time so that we
	// have an idea of when we should hear back from them - else the idle
	// timeout could fire when we were only half done sending the blocks.
	if numAdded > 0 {
		<-doneChan
	}
}

// handleGetDirBlocksMsg is invoked when a peer receives a getdirblocks factom message.
func (p *peer) handleGetDirBlocksMsg(msg *wire.MsgGetDirBlocks) {
	// Return all block hashes to the latest one (up to max per message) if
	// no stop hash was specified.
	// Attempt to find the ending index of the stop hash if specified.
	// fmt.Println("handleGetDirBlocksMsg: ", spew.Sdump(msg))
	endIdx := database.AllShas //factom db
	if !msg.HashStop.IsEqual(&zeroBtcHash) {
		height, err := db.FetchBlockHeightBySha(&msg.HashStop)
		if err == nil {
			endIdx = height + 1
		} else {
			fmt.Println("handleGetDirBlocksMsg: error in db.FetchBlockHeightBySha. ", err.Error())
		}
	}

	// Find the most recent known block based on the block locator.
	// Use the block after the genesis block if no other blocks in the
	// provided locator are known.  This does mean the client will start
	// over with the genesis block if unknown block locators are provided.
	// This mirrors the behavior in the reference implementation.
	startIdx := int64(1)
	for _, hash := range msg.BlockLocatorHashes {
		height, err := db.FetchBlockHeightBySha(hash)
		if err == nil {
			// Start with the next hash since we know this one.
			startIdx = height + 1
			break
		} else {
			fmt.Println("handleGetDirBlocksMsg: error in db.FetchBlockHeightBySha. ", err.Error())
		}

	}

	peerLog.Info("startIdx=", startIdx, ", endIdx=", endIdx)

	// Don't attempt to fetch more than we can put into a single message.
	autoContinue := false
	if endIdx-startIdx > wire.MaxBlocksPerMsg {
		endIdx = startIdx + wire.MaxBlocksPerMsg
		autoContinue = true
	}

	// Generate inventory message.
	//
	// The FetchBlockBySha call is limited to a maximum number of hashes
	// per invocation.  Since the maximum number of inventory per message
	// might be larger, call it multiple times with the appropriate indices
	// as needed.
	invMsg := wire.NewMsgDirInv()
	for start := startIdx; start < endIdx; {
		// Fetch the inventory from the block database.
		hashList, err := db.FetchHeightRange(start, endIdx)
		if err != nil {
			peerLog.Warnf("Dir Block lookup failed: %v", err)
			return
		}

		// The database did not return any further hashes.  Break out of
		// the loop now.
		if len(hashList) == 0 {
			break
		}
		peerLog.Infof("len(hashList)=%d", len(hashList))

		// Add dir block inventory to the message.
		for _, hash := range hashList {
			hashCopy := hash
			iv := wire.NewInvVect(wire.InvTypeFactomDirBlock, &hashCopy)
			invMsg.AddInvVect(iv)
		}
		start += int64(len(hashList))
	}
	peerLog.Infof("len(invList)=%d, autoContinue=%t", len(invMsg.InvList), autoContinue)

	// Send the inventory message if there is anything to send.
	if len(invMsg.InvList) > 0 {
		invListLen := len(invMsg.InvList)
		if autoContinue && invListLen == wire.MaxBlocksPerMsg {
			// Intentionally use a copy of the final hash so there
			// is not a reference into the inventory slice which
			// would prevent the entire slice from being eligible
			// for GC as soon as it's sent.
			continueHash := invMsg.InvList[invListLen-1].Hash
			p.continueHash = &continueHash
		}
		p.QueueMessage(invMsg, nil)
	}
}

// handleGetFactomDataMsg is invoked when a peer request factom data based on height.
// requested data can be: ablock, dblock, fblock, ecblok, eblock, and entries.
// since entries have no height, so all entries in one eblock of that height will be delivered.
func (p *peer) handleGetFactomDataMsg(msg *wire.MsgGetFactomData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()
	doneChan := make(chan struct{}, 1)

	for i, iv := range msg.InvList {
		var c chan struct{}
		// If this will be the last message we send.
		if i == len(msg.InvList)-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else if (i+1)%3 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}
		var err error
		switch iv.Type {
		case wire.InvTypeFactomGetDirData:	// same as MsgGetDirData
			fmt.Printf("handleGetFactomDataMsg: GetDirData: height %d or hash %s\n", iv.Height, iv.Hash)
			if iv.Hash == *zeroHash && iv.Height == -1 {
				err = fmt.Errorf("No dir data found for height %d or hash %s\n", iv.Height, iv.Hash)
			} else if iv.Hash != *zeroHash {
				err = p.pushDirBlockMsg(wire.FactomHashToShaHash(&iv.Hash), c, nil)
			} else if iv.Height > -1 {
				hash, err0 := db.FetchDBHashByHeight(uint32(iv.Height))
				if err0 != nil {
					fmt.Println("FetchDBHashByHeight err: ", err0.Error())
					err = err0
				} else {
					err = p.pushDirBlockMsg(wire.FactomHashToShaHash(hash), c, nil)
				}
			}

		case wire.InvTypeFactomDirBlock:	// get one dir block only, no other blocks after that
			fmt.Printf("handleGetFactomDataMsg: DirBlock: height %d or hash %s\n", iv.Height, iv.Hash.String())
			if iv.Hash == *zeroHash && iv.Height == -1 {
				err = fmt.Errorf("No dir block found for height %d or hash %s\n", iv.Height, iv.Hash.String())
			} else if iv.Hash != *zeroHash {
				msg := wire.NewMsgDirBlock()
				blk, err0 := db.FetchDBlockByHash(&iv.Hash)		// FetchDBlockByMR
				if err0 != nil {
					fmt.Println("FetchDBlockByMR err: ", err0.Error())
					err = err0
				} else {
					msg.DBlk = blk
					msg.NonDirBlockNeeded = false
					p.QueueMessage(msg, c) 			
				}
			} else if iv.Height > -1 {
				msg := wire.NewMsgDirBlock()
				blk, err0 := db.FetchDBlockByHeight(uint32(iv.Height))
				if err0 != nil {
					fmt.Println("FetchDBlockByHeight err: ", err0.Error())
					err = err0
				} else {
					msg.DBlk = blk
					msg.NonDirBlockNeeded = false
					p.QueueMessage(msg, c)
				}
			}

		case wire.InvTypeFactomAdminBlock:
			fmt.Printf("handleGetFactomDataMsg: AdminBlock: height %d or hash %s\n", iv.Height, iv.Hash.String())
			if iv.Hash == *zeroHash && iv.Height == -1 {
				err = fmt.Errorf("No admin block found for height %d or hash %s\n", iv.Height, iv.Hash.String())
			} else if iv.Hash != *zeroHash {
				err = p.pushABlockMsg(&iv.Hash, c, nil)
			} else if iv.Height > -1 {
				msg := wire.NewMsgABlock()
				blk, err0 := db.FetchABlockByHeight(uint32(iv.Height))
				if err0 != nil {
					fmt.Println("FetchABlockByHeight err: ", err0.Error())
					err = err0
				} else {
					msg.ABlk = blk
					p.QueueMessage(msg, c)
				}
			}
			
		case wire.InvTypeFactomEntryCreditBlock:
			fmt.Printf("handleGetFactomDataMsg: ECBlock: height %d or hash %s\n", iv.Height, iv.Hash.String())
			if iv.Hash == *zeroHash && iv.Height == -1 {
				err = fmt.Errorf("No entryCredit block found for height %d or hash %s\n", iv.Height, iv.Hash.String())
			} else if iv.Hash != *zeroHash {
				err = p.pushECBlockMsg(&iv.Hash, c, nil)
			} else if iv.Height > -1 {
				msg := wire.NewMsgECBlock()
				blk, err0 := db.FetchECBlockByHeight(uint32(iv.Height))
				if err0 != nil {
					fmt.Println("FetchECBlockByHeight err: ", err0.Error())
					err = err0
				} else {
					msg.ECBlock = blk
					p.QueueMessage(msg, c)
				}
			}
			
		case wire.InvTypeFactomFBlock:
			fmt.Printf("handleGetFactomDataMsg: FactoidBlock: height %d or hash %s\n", iv.Height, iv.Hash.String())
			if iv.Hash == *zeroHash && iv.Height == -1 {
				err = fmt.Errorf("No factoid block found for height %d or hash %s\n", iv.Height, iv.Hash.String())
			} else if iv.Hash != *zeroHash {
				err = p.pushFBlockMsg(&iv.Hash, c, nil)
			} else if iv.Height > -1 {
				msg := wire.NewMsgFBlock()
				blk, err0 := db.FetchFBlockByHeight(uint32(iv.Height))
				if err0 != nil {
					fmt.Println("FetchFBlockByHeight err: ", err0.Error())
					err = err0
				} else {
					msg.SC = blk
					p.QueueMessage(msg, c)
				}
			}
			
		case wire.InvTypeFactomEntryBlock:	// get only one entry block, no entry after that
			fmt.Printf("handleGetFactomDataMsg: EntryBlock: height %d or hash %s\n", iv.Height, iv.Hash.String())
			if iv.Hash == *zeroHash && iv.Height == -1 {
				err = fmt.Errorf("No entry block found for height %d or hash %s\n", iv.Height, iv.Hash.String())
			} else if iv.Hash != *zeroHash && iv.Height == -1 {
				msg := wire.NewMsgEBlock()
				blk, err0 := db.FetchEBlockByMR(&iv.Hash)
				if err0 != nil {
					fmt.Println("FetchEBlockByMR err: ", err0.Error())
					err = err0
				} else {
					msg.EBlk = blk
					msg.EntryNeeded = false
					p.QueueMessage(msg, c)
				}
			} else if iv.Hash != *zeroHash && iv.Height > -1 {
				msg := wire.NewMsgEBlock()
				blk, err0 := db.FetchEBlockByHeight(&iv.Hash, uint32(iv.Height))
				if err0 != nil {
					fmt.Println("FetchEBlockByHeight err: ", err0.Error())
					err = err0
				} else {
					msg.EBlk = blk
					msg.EntryNeeded = false
					p.QueueMessage(msg, c)
				}
			}
			
		case wire.InvTypeFactomEntry:
			fmt.Printf("handleGetFactomDataMsg: Entry: height %d or hash %s\n", iv.Height, iv.Hash.String())
			if iv.Hash == *zeroHash {
				err = fmt.Errorf("No entry block found for hash %s\n", iv.Hash.String())
			} else {
				msg := wire.NewMsgEntry()
				entry, err0 := db.FetchEntryByHash(&iv.Hash)
				if err0 != nil {
					fmt.Println("FetchEntryByHash err: ", err0.Error())
					err = err0
				} else {
					msg.Entry = entry
					p.QueueMessage(msg, c) 			
				}
			}

		default:
			peerLog.Warnf("Unknown type in inventory request %d",
				iv.Type)
			continue
		}
		if err != nil {
			fmt.Println("handleGetFactomDataMsg: err=", err.Error())
			ivv := wire.NewInvVect(iv.Type, wire.FactomHashToShaHash(&iv.Hash))
			notFound.AddInvVect(ivv)

			if i == len(msg.InvList)-1 && c != nil {
				<-c
			}
		}
		numAdded++
		// waitChan = c
	}
	if len(notFound.InvList) != 0 {
		p.QueueMessage(notFound, doneChan)
	}
	if numAdded > 0 {
		<-doneChan
	}
}

// pushDirBlockMsg sends a dir block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (p *peer) pushDirBlockMsg(sha *wire.ShaHash, doneChan, waitChan chan struct{}) error {
	commonhash := new(common.Hash)
	commonhash.SetBytes(sha.Bytes())
	blk, err := db.FetchDBlockByHash(commonhash)

	if err != nil {
		// check newly generated dir block in processor, 
		var hash *common.Hash
		if newDBlock != nil {
			bin, _ := newDBlock.MarshalBinary()
			hash = common.Sha(bin)
		}
		if hash != nil && bytes.Compare(commonhash.Bytes(), hash.Bytes()) == 0 {
			fmt.Println("found dirBlock as the newly generated dir block: ", 
				newDBlock.Header.DBHeight)
			blk = newDBlock
		} else {
			peerLog.Tracef("Unable to fetch requested dir block sha %v: %v", 
				sha, err)

			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return err
		}
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	// We only send the channel for this message if we aren't sending(sha)
	// an inv straight after.
	var dc chan struct{}
	sendInv := p.continueHash != nil && p.continueHash.IsEqual(sha)
	if !sendInv {
		dc = doneChan
	}
	msg := wire.NewMsgDirBlock()
	msg.DBlk = blk
	p.QueueMessage(msg, dc) //blk.MsgBlock(), dc)

	// When the peer requests the final block that was advertised in
	// response to a getblocks message which requested more blocks than
	// would fit into a single message, send it a new inventory message
	// to trigger it to issue another getblocks message for the next
	// batch of inventory.
	if p.continueHash != nil && p.continueHash.IsEqual(sha) {
		peerLog.Debug("continueHash: " + sha.String())
		// Sleep for 5 seconds for the peer to catch up
		time.Sleep(5 * time.Second)

		//
		// Note: Rather than the latest block height, we should pass
		// the last block height of this batch of wire.MaxBlockLocatorsPerMsg
		// to signal this is the end of the batch and
		// to trigger a client to send a new GetDirBlocks message
		//
		//hash, _, err := db.FetchBlockHeightCache()
		//if err == nil {
		invMsg := wire.NewMsgDirInvSizeHint(1)
		iv := wire.NewInvVect(wire.InvTypeFactomDirBlock, sha) //hash)
		invMsg.AddInvVect(iv)
		p.QueueMessage(invMsg, doneChan)
		p.continueHash = nil
		//} else if doneChan != nil {
		if doneChan != nil {
			doneChan <- struct{}{}
		}
	}
	return nil
}

// PushGetDirBlocksMsg sends a getdirblocks message for the provided block locator
// and stop hash.  It will ignore back-to-back duplicate requests.
func (p *peer) PushGetDirBlocksMsg(locator BlockLocator, stopHash *wire.ShaHash) error {

	// Extract the begin hash from the block locator, if one was specified,
	// to use for filtering duplicate getblocks requests.
	// request.
	var beginHash *wire.ShaHash
	if len(locator) > 0 {
		beginHash = locator[0]
	}

	// Filter duplicate getdirblocks requests.
	if p.prevGetBlocksStop != nil && p.prevGetBlocksBegin != nil &&
		beginHash != nil && stopHash.IsEqual(p.prevGetBlocksStop) &&
		beginHash.IsEqual(p.prevGetBlocksBegin) {

		peerLog.Tracef("Filtering duplicate [getdirblocks] with begin "+
			"hash %v, stop hash %v", beginHash, stopHash)
		return nil
	}

	// Construct the getblocks request and queue it to be sent.
	msg := wire.NewMsgGetDirBlocks(stopHash)
	for _, hash := range locator {
		err := msg.AddBlockLocatorHash(hash)
		if err != nil {
			return err
		}
	}
	p.QueueMessage(msg, nil)

	// Update the previous getblocks request information for filtering
	// duplicates.
	p.prevGetBlocksBegin = beginHash
	p.prevGetBlocksStop = stopHash
	return nil
}

// pushGetNonDirDataMsg takes the passed DBlock
// and return corresponding data block like Factoid block,
// EC block, Entry block, and Entry
func (p *peer) pushGetNonDirDataMsg(dblock *common.DirectoryBlock) {
	binary, _ := dblock.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())

	iv := wire.NewInvVect(wire.InvTypeFactomNonDirBlock, hash)
	gdmsg := wire.NewMsgGetNonDirData()
	gdmsg.AddInvVect(iv)
	if len(gdmsg.InvList) > 0 {
		p.QueueMessage(gdmsg, nil)
	}
}

// pushGetEntryDataMsg takes the passed EBlock
// and return all the corresponding EBEntries
func (p *peer) pushGetEntryDataMsg(eblock *common.EBlock) {
	binary, _ := eblock.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())

	iv := wire.NewInvVect(wire.InvTypeFactomEntry, hash)
	gdmsg := wire.NewMsgGetEntryData()
	gdmsg.AddInvVect(iv)
	if len(gdmsg.InvList) > 0 {
		p.QueueMessage(gdmsg, nil)
	}
}

// pushFBlockMsg sends an factoid block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (p *peer) pushFBlockMsg(commonhash *common.Hash, doneChan, waitChan chan struct{}) error {
	blk, err := db.FetchFBlockByHash(commonhash)

	if err != nil || blk == nil {
		// check newly generated factoid block in processor, 
		var hash factoid.IHash
		if newFBlock != nil {
			hash = newFBlock.GetHash()
		}
		if hash != nil && bytes.Compare(commonhash.Bytes(), hash.Bytes()) == 0 {
			fmt.Println("found it as the newly generated factoid Block: ", newFBlock.GetDBHeight())
			blk = newFBlock
		} else {
			peerLog.Tracef("Unable to fetch requested factoid block sha %v: %v",
				commonhash, err)

			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return err
		}
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	msg := wire.NewMsgFBlock()
	msg.SC = blk
	p.QueueMessage(msg, doneChan) //blk.MsgBlock(), dc)
	return nil
}

// pushABlockMsg sends an admin block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (p *peer) pushABlockMsg(commonhash *common.Hash, doneChan, waitChan chan struct{}) error {
	blk, err := db.FetchABlockByHash(commonhash)

	if err != nil || blk == nil {
		// check newly generated admin block in processor, 
		var hash *common.Hash
		if newABlock != nil {
			hash, _ = newABlock.PartialHash()
		}
		if hash != nil && bytes.Compare(commonhash.Bytes(), hash.Bytes()) == 0 {
			fmt.Println("found it as the newly generated admin Block: ", 
				newABlock.Header.DBHeight)
			blk = newABlock
		} else {
			peerLog.Tracef("Unable to fetch requested Admin block sha %v: %v",
				commonhash, err)
			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return err
		}
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	msg := wire.NewMsgABlock()
	msg.ABlk = blk
	p.QueueMessage(msg, doneChan) //blk.MsgBlock(), dc)
	return nil
}

// pushECBlockMsg sends a entry credit block message for the provided block
// hash to the connected peer.  An error is returned if the block hash is not
// known.
func (p *peer) pushECBlockMsg(commonhash *common.Hash, doneChan, waitChan chan struct{}) error {
	blk, err := db.FetchECBlockByHash(commonhash)
	if err != nil || blk == nil {
		// check newly generated Entry Credit block in processor, 
		var hash *common.Hash
		if newECBlock != nil {
			hash, _ = newECBlock.HeaderHash()
		}
		if hash != nil && bytes.Compare(commonhash.Bytes(), hash.Bytes()) == 0 {
			fmt.Println("found it as the newly generated Entry Credit Block: ", 
				newECBlock.Header.EBHeight)
			blk = newECBlock
		} else {
			peerLog.Tracef("Unable to fetch requested Entry Credit block sha %v: %v",
				commonhash, err)

			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return err
		}
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	msg := wire.NewMsgECBlock()
	msg.ECBlock = blk
	p.QueueMessage(msg, doneChan) //blk.MsgBlock(), dc)
	return nil
}

// pushEBlockMsg sends a entry block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (p *peer) pushEBlockMsg(commonhash *common.Hash, doneChan, waitChan chan struct{}) error {
	blk, err := db.FetchEBlockByMR(commonhash)
	if err != nil {
		// check newly generated Entry block in processor, 
		for _, eblock := range newEBlocks {
			if eblock == nil {
				continue
			}
			hash, _ := eblock.Hash()
			if hash != nil && bytes.Compare(commonhash.Bytes(), hash.Bytes()) == 0 {
				fmt.Println("found it as the newly generated Entry Block: ", 
					eblock.Header.EBHeight)
				blk = eblock
				err = nil
				break
			}
		}
		if err != nil || blk == nil {
			peerLog.Tracef("Unable to fetch requested Entry block sha %v: %v",
				commonhash, err)
				
			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return err
		}
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	msg := wire.NewMsgEBlock()
	msg.EBlk = blk
	p.QueueMessage(msg, doneChan) //blk.MsgBlock(), dc)
	return nil
}

// pushEntryMsg sends a EBlock entry message for the provided ebentry hash to the
// connected peer.  An error is returned if the block hash is not known.
func (p *peer) pushEntryMsg(commonhash *common.Hash, doneChan, waitChan chan struct{}) error {
	entry, err := db.FetchEntryByHash(commonhash)
	if err != nil || entry == nil {
		peerLog.Tracef("Unable to fetch requested Entry sha %v: %v",
			commonhash, err)
		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	msg := wire.NewMsgEntry()
	msg.Entry = entry
	p.QueueMessage(msg, doneChan) //blk.MsgBlock(), dc)
	return nil
}

// handleFactoidMsg
func (p *peer) handleFactoidMsg(msg *wire.MsgFactoidTX, buf []byte) {
	// the factoid balance should be update during syncup
	// fmt.Println("handleFactoidMsg: ", spew.Sdump(msg))
	if ClientOnly {
		return
	}

	binary, _ := msg.Transaction.MarshalBinary()
	commonHash := common.Sha(binary)
	hash, _ := wire.NewShaHash(commonHash.Bytes())

	iv := wire.NewInvVect(wire.InvTypeTx, hash)
	p.AddKnownInventory(iv)

	inMsgQueue <- msg
}

// Handle factom app imcoming msg
func (p *peer) handleCommitChainMsg(msg *wire.MsgCommitChain) {
	if !ClientOnly {
		inMsgQueue <- msg
	}
}

// Handle factom app imcoming msg
func (p *peer) handleCommitEntryMsg(msg *wire.MsgCommitEntry) {
	if !ClientOnly {
		inMsgQueue <- msg
	}
}

// Handle factom app imcoming msg
func (p *peer) handleRevealEntryMsg(msg *wire.MsgRevealEntry) {
	if !ClientOnly {
		inMsgQueue <- msg
	}
}

// Handle factom app imcoming msg
func (p *peer) handleAckMsg(msg *wire.MsgAck) {
	if !ClientOnly {
		p.server.blockManager.QueueAck(msg, p)
	}

	// this is needed only when i missed the currentLeader msg
	// I have no other way to know who the leader is except through ack
	leader := p.server.GetLeaderPeer()
	//fmt.Printf("handleAckMsg: current Leader Peer:%s, Ack coming from peer: %s\n", leader, p)
	if p != leader && p.isFederateServer() {
		fmt.Println("handleAckMsg: RESET Leader Peer to ", p)
		if leader != nil {
			fs := p.server.GetFederateServer(leader)
			if fs != nil {
				fs.LeaderLast = msg.Height
			}
		}
		p.server.SetLeaderPeer(p)
		p.server.latestLeaderSwitchDBHeight = msg.Height
	}
}

func (p *peer) isFederateServer() bool {
	for _, fed := range p.server.federateServers {
		if fed.Peer.nodeID == p.nodeID && fed.Peer.id == p.id {
			//fmt.Printf("It's a federate server: %s\n", fed.Peer)
			return true
		}
	}
	return false
}

// Handle factom app imcoming msg
func (p *peer) handleDirBlockSigMsg(msg *wire.MsgDirBlockSig) {
	// Add the msg to inbound msg queue
	if !ClientOnly {
		inMsgQueue <- msg
	}
}

// returns true if the message should be relayed, false otherwise
func (p *peer) shallRelay(msg interface{}) bool {
	hash, _ := wire.NewShaHashFromStruct(msg)
	iv := wire.NewInvVect(wire.InvTypeFactomRaw, hash)
	if !p.isKnownInventory(iv) {
		p.AddKnownInventory(iv)
		return true
	}
	fmt.Println("******************* SHALL NOT RELAY !!!!!!!!!!! ******************")
	return false
}

// FactomRelay Calls FactomRelay to relay/broadcast a Factom message (to your peers).
// The intent is to call this function after certain 'processor' checks been done.
func (p *peer) FactomRelay(msg wire.Message) {
	// broadcast/relay only if hadn't been done for this peer
	if p.shallRelay(msg) {
		p.server.BroadcastMessage(msg)
	}
}

//GetNodeID returns this peer's nodeID
func (p *peer) GetNodeID() string {
	return p.nodeID
}

func (p *peer) handleNextLeaderMsg(msg *wire.MsgNextLeader) {
	if ClientOnly {
		return
	}
	fmt.Printf("handleNextLeaderMsg: %s\n", msg)
	if !msg.Sig.Verify([]byte(msg.CurrLeaderID+msg.NextLeaderID)) {
		fmt.Println("handleNextLeaderMsg: signature verify FAILED.")
		return
	}
	// leader := p.server.GetLeaderPeer()
	// fmt.Printf("leader=%s, p=%s, my.fs.p=%s, p==fs.p: %t, leader==p: %t\n", leader, p, 
		// p.server.GetMyFederateServer().Peer, p.server.GetMyFederateServer().Peer==p, leader==p)
		
	if p.nodeID != msg.CurrLeaderID {
		fmt.Printf("handleNextLeaderMsg: leader verify FAILED: my leader is %s, but msg.leader is %s\n",
			p.nodeID, msg.CurrLeaderID)
		return
	}
	if p.server.nodeID == msg.NextLeaderID {
		policy := &leaderPolicy{
			NextLeader:     p.server.GetMyFederateServer().Peer,
			StartDBHeight:  msg.StartDBHeight,
			NotifyDBHeight: defaultNotifyDBHeight,
			Term:           defaultLeaderTerm,
		}
		p.server.myLeaderPolicy = policy
		p.server.setLeaderElectByID(p.server.nodeID)
		fmt.Println("handleNextLeaderMsg: I'm the next leader elected. my.fs=", 
			p.server.GetMyFederateServer().Peer)

		sig := p.server.privKey.Sign([]byte(msg.CurrLeaderID + msg.NextLeaderID))
		resp := wire.NewNextLeaderRespMsg(msg.CurrLeaderID, msg.NextLeaderID,
			msg.StartDBHeight, sig, true)
		fmt.Printf("handleNextLeaderMsg: sending NextLeaderRespMsg, leaderElected=%t\n", p.server.IsLeaderElect())
		p.server.BroadcastMessage(resp)
	}
}

func (p *peer) handleNextLeaderRespMsg(msg *wire.MsgNextLeaderResp) {
	if ClientOnly {
		return
	}
	fmt.Printf("handleNextLeaderRespMsg: %s\n", msg)
	if !msg.Sig.Verify([]byte(msg.CurrLeaderID+msg.NextLeaderID)) {
		fmt.Println("handleNextLeaderRespMsg: signature verify FAILED.")
		return
	}
	if p.server.IsLeader() && p.server.nodeID == msg.CurrLeaderID {
		p.server.myLeaderPolicy.Confirmed = true
	}
	p.nodeState = wire.NodeLeaderElect 
	fmt.Printf("handleNextLeaderRespMsg: next leader CONFIRMED: %s. startHeight=%d, p=%s\n", 
		msg.NextLeaderID, msg.StartDBHeight, p)
}

// MsgMissing is triggered by ack from leader. So this peer/server has to be leader
func (p *peer) handleMissingMsg(msg *wire.MsgMissing) {
	if ClientOnly {
		return
	}
	fmt.Printf("handleMissingMsg: %s\n", msg)
	if !p.server.IsLeader() {
		fmt.Println("handleMissingMsg: MsgMissing has to be handled by the leader")
		return
	}
	if !msg.Sig.Verify(msg.GetBinaryForSignature()) {
		fmt.Println("handleMissingMsg: could not verify sig: ", msg)
		return
	}
	m := plMgr.MyProcessList.GetMissingMsg(msg)
	if m == nil {
		fmt.Println("handleMissingMsg: found no msg/ack it in process list.")
		if !msg.IsAck {
			m = fMemPool.getMsg(msg.ShaHash)
		}
	}
	if m == nil {
		notFound := wire.NewMsgNotFound()
		iv := wire.NewInvVect(wire.InvType(msg.Type), &msg.ShaHash)
		notFound.AddInvVect(iv)
		p.QueueMessage(notFound, nil)
		return
	}
	fmt.Printf("handleMissingMsg: found missing msg and sending it: %s\n", m)
	p.QueueMessage(m, nil)
}

func (p *peer) handleCandidateMsg(msg *wire.MsgCandidate) {
	if ClientOnly {
		return
	}
	fmt.Printf("handleCandidateMsg: %s\n", msg)
	if !msg.Sig.Verify([]byte(string(msg.DBHeight) + msg.SourceNodeID)) {
		fmt.Println("handleCandidateMsg: signature verify FAILED.")
		return
	}
	if p.nodeID == msg.SourceNodeID { 
		p.nodeState = wire.NodeFollower
		fs := p.server.GetFederateServer(p)
		if fs != nil {
			fs.FirstAsFollower = msg.DBHeight
		}
		// fmt.Println("handleCandidateMsg: candidate turned to follower: ", spew.Sdump(fs))
	}
}

func (p *peer) handleCurrentLeaderMsg(msg *wire.MsgCurrentLeader) {
	if ClientOnly {
		return
	}
	fmt.Printf("handleCurrentLeaderMsg: %s\n", msg)
	// The timing b/w leader swtich in server.handleNextLeader and here could vary.
	// So no need to check if p.server.IsLeader()	
	// if p.server.IsLeader() {
		// panic("handleCurrentLeaderMsg: I'm the current leader, no need to select a new current leader")
	// }
	if p.server.IsLeaderElect() {
		panic("handleCurrentLeaderMsg: i'm the leaderElect, but new current leader is " + msg.NewLeaderCandidates)
	}
	if !msg.Sig.Verify([]byte(msg.CurrLeaderGone + msg.NewLeaderCandidates + msg.SourceNodeID + strconv.Itoa(int(msg.StartDBHeight)))) {
		panic("handleCurrentLeaderMsg: signature verify FAILED.")
	}
	
	// in normal regime change, when handleAckMsg come before this msg, and changed peer state
	fmt.Println("handleCurrentLeaderMsg: ", spew.Sdump(p.server.federateServers))
	prev := p.server.GetPrevLeaderPeer()
	curr := p.server.GetLeaderPeer()
	if curr != nil && curr.nodeID == msg.NewLeaderCandidates &&
		prev != nil && prev.nodeID == msg.CurrLeaderGone {
		fmt.Printf("handleCurrentLeaderMsg: normal regime change done already. do nothing here. curr=%s. prev=%s\n", curr, prev)
		return
	}
	
	/*
	// the timing between resetting and verifying the currentLeader could be tricky.
	// it's uncertain which one is done first. So omit this step 
	if !(p.server.leaderPeer != nil && p.server.leaderPeer.nodeID == msg.CurrLeaderGone) {
		s := fmt.Sprintf("handleCurrentLeaderMsg: leader verify FAILED: my leader is %s, but msg.leader is %s\n",
			p.server.leaderPeer.nodeID, msg.CurrLeaderGone)
		//return
		panic(s)
	}
	// this could happen when a new block is downloaded but not saved to db yet
	// when a candidate is just turned to a follower
	_, newestHeight, _ := db.FetchBlockHeightCache()
	if uint32(newestHeight) != msg.StartDBHeight-1 {
		s := fmt.Sprintf("handleNextLeaderMsg: my DBHeight=%d, msg.StartDBHeight=%d\n", 
			newestHeight, msg.StartDBHeight)
		//return
		panic(s)
	}*/

	// verify the new leader to see if it follows the rule
	// 0). if it's the only follower left, it's the new leader
	// 1). if leaderElect exists, it's the new leader
	// 2). else if prev leader exists, it's the new leader
	// 3). else it's the peer with the longest FirstJoined
	// var next *federateServer
	nonCandidates, _ := p.server.nonCandidateServers()
	
	/*
	// this case will be handled by the longest tenured case 
	// check if it's the only follower left
	if !p.IsCandidate() && len(nonCandidates) == 1 && nonCandidates[0].Peer.nodeID == msg.NewLeaderCandidates {
		fmt.Println("handleCurrentLeaderMsg: It's the only follower left. ", msg.NewLeaderCandidates)
		p.resetFollowerState(nonCandidates[0].Peer, msg)
		return
	}*/
			
	// find the leaderElect, excluding candidates
	next := p.server.GetLeaderElect()		
	if next != nil {
		fmt.Printf("handleCurrentLeaderMsg: leaderElect: %s, p=%s\n", next, p)
		// the timing of udpate leader status could be different
		// let stop checking to avoid inconsistancy
		if next.nodeID != msg.NewLeaderCandidates {
			panic("handleCurrentLeaderMsg: the leaderElect is " + next.nodeID + 
				", but new current leader is " + msg.NewLeaderCandidates)
		} else {	// next should be p
			fmt.Println("handleCurrentLeaderMsg: the leaderElect is the new leader: " + msg.NewLeaderCandidates)
			p.resetFollowerState(next, msg)
			return
		}
	} 
	
	// find the prev leader
	if prev != nil {
		fmt.Printf("handleCurrentLeaderMsg: prev leader: %s, p=%s\n", prev, p)
		if prev.nodeID != msg.NewLeaderCandidates {
			panic("handleCurrentLeaderMsg: the prev peer is " + prev.nodeID + 
				", but new current leader is " + msg.NewLeaderCandidates)
		} else {
			fmt.Println("handleCurrentLeaderMsg: the prev leader is the new leader: " + msg.NewLeaderCandidates)
			p.resetFollowerState(prev, msg)
			return
		}
	} 
	// find out the server with the longest tenure
	sort.Sort(ByStartTime(nonCandidates))
	if nonCandidates[0].Peer.nodeID == msg.NewLeaderCandidates {
		fmt.Printf("handleCurrentLeaderMsg: longest FirstJoined: %s, p=%s\n", nonCandidates[0].Peer, p)
		p.resetFollowerState(nonCandidates[0].Peer, msg)
		return
	}
	panic("no such candidate for current new leader.")
}

func (p *peer) resetFollowerState(leader *peer, msg *wire.MsgCurrentLeader) {
	fs := p.server.GetFederateServerByID(msg.CurrLeaderGone)
	if fs != nil {
		fs.LeaderLast = msg.StartDBHeight - 1
	}
	p.server.SetLeaderPeer(leader)
	p.server.latestLeaderSwitchDBHeight = msg.StartDBHeight
	fmt.Printf("resetFollowerState: leader=%s, fs=%s\n", leader, spew.Sdump(p.server.federateServers))

	// reset eCreditMap & chainIDMap & processList
	if leaderCrashed {
		eCreditMap = eCreditMapBackup
		chainIDMap = chainIDMapBackup
		commitChainMap = commitChainMapBackup
		commitEntryMap = commitEntryMapBackup
		initProcessListMgr()
	}
}

func (p *peer) IsCandidate() bool {
	return p.nodeState == wire.NodeCandidate
}

func (p *peer) IsFollower() bool {
	return p.nodeState == wire.NodeFollower || p.nodeState == wire.NodeLeaderElect || 
		p.nodeState == wire.NodeLeaderPrev
}

func (p *peer) IsLeaderElect() bool {
	return p.nodeState == wire.NodeLeaderElect
}

func (p *peer) IsLeader() bool {
	return p.nodeState == wire.NodeLeader
}

func (p *peer) IsPrevLeader() bool {
	return p.nodeState == wire.NodeLeaderPrev
}
