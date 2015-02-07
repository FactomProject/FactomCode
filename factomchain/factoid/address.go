// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"github.com/FactomProject/FactomCode/notaryapi"	
	"github.com/FactomProject/btcutil/base58"
)

//raw address, either a hash of *Reveal (for factoid tx) 
//	or a raw PublicKey (for entrycredit tx)
type Address []byte

//revealed address is the public key
type AddressReveal	notaryapi.DetachedPublicKey

//multisig reveal structure that hashes into raw Address 
type MultisigReveal struct {
	NumRequired				uint8
	Addresses 				[]AddressReveal
}  

//EntryCredit transactions sent directly to publickey
type EntryCreditAddress AddressReveal

//single signature of tx,
//	Hint is not signed! is used to help match sig to address in multisig input.
type SingleSignature struct {
	Hint				byte
	Sig 				notaryapi.DetachedSignature
}

//all sigs needed for input
//	Hint is used to help match sig to multi input tx 
//		signatures can then be reordered correctly, so can be 
//		vefified w/o hint. this allows signatures to sent and 
//		attached to tx in any order 
type InputSig struct {
	Hint				byte
	Sigs 				[]SingleSignature
}


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

