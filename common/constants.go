// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"fmt"
	"encoding/json"
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

	
	// maxProtocolVersion is the max protocol version the peer supports.
	//Common constants
	VERSION_0         = byte(0)
	FACTOMD_VERSION   = 3004               //fixed point. resolves to 0.<thousands place>.<rightmost digits>
	NETWORK_ID_DB 	  = uint32(4203931041) //0xFA92E5A1
	NETWORK_ID_EB     = uint32(4203931042) //0xFA92E5A2
	NETWORK_ID_CB     = uint32(4203931043) //0xFA92E5A3

	//For Factom TestNet
	NETWORK_ID_TEST = uint32(0) //0x0

	//Server running mode
	FULL_NODE   = "FULL"
	SERVER_NODE = "SERVER"
	LIGHT_NODE  = "LIGHT"

	//Server public key for milestone 1
	SERVER_PUB_KEY         = "0426a802617848d4d16d87830fc521f4d136bb2d0c352850919c2679f189613a"
	//Genesis directory block timestamp in RFC3339 format
	GENESIS_BLK_TIMESTAMP = "2015-09-01T20:00:00+00:00"
	//Genesis directory block hash
	GENESIS_DIR_BLOCK_HASH = "cbd3d09db6defdc25dfc7d57f3479b339a077183cd67022e6d1ef6c041522b40"

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

// Structure for reporting properties (used by the web API
//
type Properties struct {
	Protocol_Version  int
	Factomd_Version	  int
	Fctwallet_Version int
}

func (p *Properties) MarshalJSON() ([]byte, error) {
	type tmp struct {
		Protocol_Version  string
		Factomd_Version	  string
		Fctwallet_Version string
	}
	t := new(tmp)
	
	t.Protocol_Version = versionToString(p.Protocol_Version)
	t.Factomd_Version = versionToString(p.Factomd_Version)
	t.Fctwallet_Version = versionToString(p.Fctwallet_Version)
	
	return json.Marshal(t)
}

// versionToString converts the fixed poit versions to human readable version
// strings.
func versionToString(f int) string {
	v1 := f / 1000000
	v2 := (f % 1000000) / 1000
	v3 := f % 1000
	
	return fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}
