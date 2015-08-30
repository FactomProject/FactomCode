// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"time"
	)

const (

	//Entry Credit Blocks (For now, everyone gets the same cap)
	EC_CAP = 5      //Number of ECBlocks we start with.
	AB_CAP = EC_CAP //Administrative Block Cap for AB messages

	//Limits and Sizes
	MAX_ENTRY_SIZE    = uint16(10240) //Maximum size for Entry External IDs and the Data
	HASH_LENGTH       = int(32)       //Length of a Hash
	SIG_LENGTH        = int(64)       //Length of a signature
	MAX_ORPHAN_SIZE   = int(5000)     //Prphan mem pool size
	MAX_TX_POOL_SIZE  = int(50000)    //Transaction mem pool size
	MAX_BLK_POOL_SIZE = int(500000)   //Block mem bool size
	MAX_PLIST_SIZE    = int(150000)   //MY Process List size
	
	MAX_ENTRY_CREDITS = uint8(10)	  //Max number of entry credits per entry
	MAX_CHAIN_CREDITS = uint8(20)	  //Max number of entry credits per chain
	
	COMMIT_TIME_WINDOW = time.Duration(12)	  //Time windows for commit chain and commit entry +/- 12 hours

	//Common constants
	VERSION_0     = byte(0)
	NETWORK_ID_DB = uint32(4203931041) //0xFA92E5A1
	NETWORK_ID_EB = uint32(4203931042) //0xFA92E5A2
	NETWORK_ID_CB = uint32(4203931043) //0xFA92E5A3

	//For Factom TestNet
	NETWORK_ID_TEST = uint32(0) //0x0

	//Server running mode
	FULL_NODE   = "FULL"
	SERVER_NODE = "SERVER"
	LIGHT_NODE  = "LIGHT"

	//Server public key for milestone 1
	SERVER_PUB_KEY         = "8cee85c62a9e48039d4ac294da97943c2001be1539809ea5f54721f0c5477a0a"
	//Genesis directory block timestamp in RFC3339 format
	GENESIS_BLK_TIMESTAMP = "2015-09-01T18:00:00+00:00"
	//Genesis directory block hash
	GENESIS_DIR_BLOCK_HASH = "97e2369dd8aed404205c7fb3d88538f27cc58a3293de822f037900dfdfa77a12"

)

//---------------------------------------------------------------
// Types of entries (transactions) for Admin Block
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#adminid-bytes
//---------------------------------------------------------------
const (
	TYPE_MINUTE_NUM uint8 = iota
	TYPE_DB_SIGNATURE
	TYPE_REVEAL_MATRYOSHKA
	TYPE_ADD_MATRYOSHKA
	TYPE_ADD_SERVER_COUNT
	TYPE_ADD_FED_SERVER
	TYPE_REMOVE_FED_SERVER
	TYPE_ADD_FED_SERVER_KEY
	TYPE_ADD_BTC_ANCHOR_KEY //8
)

// Chain Values.  Not exactly constants, but nice to have.
// Entry Credit Chain
var EC_CHAINID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x0c}

// Directory Chain
var D_CHAINID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x0d}

// Directory Chain
var ADMIN_CHAINID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x0a}

// Factoid chain
var FACTOID_CHAINID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x0f}

var ZERO_HASH = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
