// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"github.com/FactomProject/FactomCode/notaryapi"	
	"github.com/btcsuite/btcutil/base58"
)

type SingleSignature notaryapi.DetachedSignature

type MultiSigStruct struct {
	RequiredSigs		uint8
	Sigs 				[]SingleSignature
}  


//EntryCredit transactions sned directly to publickey
type EntryCreditAddress notaryapi.DetachedPublicKey

/*
-----------------------------------------
| netID | pubkey or hash |	checksum|	|
| 1byte	|     32 bytes   |		    |	|
-----------------------------------------
*/

// encodeAddress returns a human-readable payment address given a 32 bytes hash or 
// publike-key and netID which encodes the factom network and address type.  It is used
// in both entrycredit and factoid transactions. 
func EncodeAddress(hashokey []byte, netID byte) string {
	// Format is 1 byte for a network and address class 
	// 32 bytes for a SHA256 hash or raw PublicKey, 
	// and 4 bytes of checksum.
	return base58.CheckEncode(hashokey, netID)
}

/*
// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type.
//
// The netID network the address is associated with is extracted if possible.
// When the address does not encode the network, such as in the case of a raw
// public key, the address will be associated with the passed defaultNet.
func DecodeAddress(addr string) (hashokey []byte, error) {
	// Serialized public keys are either 65 bytes (130 hex chars) if
	// uncompressed/hybrid or 33 bytes (66 hex chars) if compressed.
	if len(addr) == 50 || len(addr) == 51 {
		hashokey, err := hex.DecodeString(addr)
		if err != nil {
			return nil, err
		}
	}
}

*/