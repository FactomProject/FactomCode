// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factomwire

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// ProtocolVersion is the latest protocol version this package supports.
	ProtocolVersion uint32 = 70002

	// MultipleAddressVersion is the protocol version which added multiple
	// addresses per message (pver >= MultipleAddressVersion).
	MultipleAddressVersion uint32 = 209

	// NetAddressTimeVersion is the protocol version which added the
	// timestamp field (pver >= NetAddressTimeVersion).
	NetAddressTimeVersion uint32 = 31402

	// RejectVersion is the protocol version which added a new reject
	// message.
	RejectVersion uint32 = 70002
)

// ServiceFlag identifies services supported by a bitcoin peer.
type ServiceFlag uint64

const (
	// SFNodeNetwork is a flag used to indicate a peer is a full node.
	SFNodeNetwork ServiceFlag = 1 << iota
)

// Map of service flags back to their constant names for pretty printing.
var sfStrings = map[ServiceFlag]string{
	SFNodeNetwork: "SFNodeNetwork",
}

// String returns the ServiceFlag in human-readable form.
func (f ServiceFlag) String() string {
	// No flags are set.
	if f == 0 {
		return "0x0"
	}

	// Add individual bit flags.
	s := ""
	for flag, name := range sfStrings {
		if f&flag == flag {
			s += name + "|"
			f -= flag
		}
	}

	// Add any remaining flags which aren't accounted for as hex.
	s = strings.TrimRight(s, "|")
	if f != 0 {
		s += "|0x" + strconv.FormatUint(uint64(f), 16)
	}
	s = strings.TrimLeft(s, "|")
	return s
}

// FactomNet represents which bitcoin network a message belongs to.
type FactomNet uint32

// Constants used to indicate the message bitcoin network.  They can also be
// used to seek to the next message when a stream's state is unknown, but
// this package does not provide that functionality since it's generally a
// better idea to simply disconnect clients that are misbehaving over TCP.
const (
	// MainNet represents the main bitcoin network.
	MainNet FactomNet = 0xd9b4bef9

	// TestNet represents the regression test network.
	TestNet FactomNet = 0xdab5bffa

	// TestNet3 represents the test network (version 3).
	TestNet3 FactomNet = 0x0709110b

	// SimNet represents the simulation test network.
	SimNet FactomNet = 0x12141c16
)

// bnStrings is a map of bitcoin networks back to their constant names for
// pretty printing.
var bnStrings = map[FactomNet]string{
	MainNet:  "MainNet",
	TestNet:  "TestNet",
	TestNet3: "TestNet3",
	SimNet:   "SimNet",
}

// String returns the FactomNet in human-readable form.
func (n FactomNet) String() string {
	if s, ok := bnStrings[n]; ok {
		return s
	}

	return fmt.Sprintf("Unknown FactomNet (%d)", uint32(n))
}
