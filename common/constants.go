// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import ()

const (

	//Entry Credit Blocks
	EC_CAP = 5 //Number of ECBlocks we start with.

	//Limits and Sizes
	MAX_ENTRY_SIZE = uint16(10240) //Maximum size for Entry External IDs and the Data
	HASH_LENGTH    = int(32)       //Length of a Hash
	//Common constants
	VERSION_0     = byte(0)
	NETWORK_ID_DB = uint32(4203931041) //0xFA92E5A1
	NETWORK_ID_EB = uint32(4203931042) //0xFA92E5A2
	NETWORK_ID_CB = uint32(4203931043) //0xFA92E5A3

	//For Factom TestNet
	NETWORK_ID_TEST = uint32(0) //0x0
)

//---------------------------------------------------------------
// Three types of entries (transactions) for Entry Credit Block
//---------------------------------------------------------------
const (
	TYPE_SERVER_INDEX uint8 = iota
	TYPE_MINUTE_NUMBER
	TYPE_PAY_CHAIN
	TYPE_PAY_ENTRY
	TYPE_BUY
)

// Chain Values.  Not exactly constants, but nice to have.
var EC_CHAINID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x0c}
