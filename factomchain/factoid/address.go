// Copyright (c) 2014 FactomProject/FactomCode Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package factoid

import (
	"bytes"
	"encoding/binary"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/btcutil/base58"
	//"strconv"
	//"errors"
)

//raw address, either a hash of *Reveal (for factoid tx)
//	or a raw PublicKey (for entrycredit tx)
type Address notaryapi.ByteArray

//revealed address is the public key
type AddressReveal notaryapi.DetachedPublicKey

//multisig reveal structure that hashes into raw Address
type MultisigReveal struct {
	NumRequired uint8
	Addresses   []AddressReveal
}

//EntryCredit transactions sent directly to publickey
type EntryCreditAddress AddressReveal

//single signature of tx,
//	Hint is not signed! is used to help match sig to address in multisig input.
type SingleSignature struct {
	Hint rune
	Sig  notaryapi.DetachedSignature
}

func (s *SingleSignature) String() string {
	return string(s.Hint) + " " + s.Sig.String()
}

func NewSingleSignature(insig *notaryapi.DetachedSignature) (ss *SingleSignature) {
	return &SingleSignature{
		Hint: 0,
		Sig:  *insig,
	}
}

func SingleSigFromByte(sb []byte) (sig SingleSignature) {
	copy(sig.Sig[:], sb)
	return
}

//all sigs needed for input
//	Hint is used to help match sig to multi input tx
//		signatures can then be reordered correctly, so can be
//		vefified w/o hint. this allows signatures to sent and
//		attached to tx in any order
type InputSig struct {
	Hint rune
	Sigs []SingleSignature
}

//AddSig to append singlesignature to InputSig
func (ins *InputSig) AddSig(ss SingleSignature) {
	ins.Sigs = append(ins.Sigs, ss)
}

func (s *SingleSignature) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, s.Hint) //int32
	buf.Write(s.Sig[:])                          // 64

	return buf.Bytes(), err
}

func (s *SingleSignature) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 4 + 64 //68
	return size
}

func (s *SingleSignature) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewReader(data[:4])
	binary.Read(buf, binary.BigEndian, &s.Hint)
	data = data[4:]

	copy(s.Sig[:], data[:64])

	return nil
}

func (is *InputSig) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, is.Hint)              //int32
	binary.Write(&buf, binary.BigEndian, uint64(len(is.Sigs))) //uint64  8
	for _, s := range is.Sigs {
		data, err = s.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	return buf.Bytes(), err
}

func (is *InputSig) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 4 + 8
	size += 68 * uint64(len(is.Sigs))
	return size
}

func (is *InputSig) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewReader(data[:4])
	binary.Read(buf, binary.BigEndian, &is.Hint)
	data = data[4:]

	count := binary.BigEndian.Uint64(data[0:8])
	data = data[8:]
	is.Sigs = make([]SingleSignature, count)
	for i := uint64(0); i < count; i++ {
		is.Sigs[i].UnmarshalBinary(data)
		data = data[68:]
	}

	return nil
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
func EncodeAddress(hashokey Address, netID byte) string {
	// Format is 1 byte for a network and address class
	// 32 bytes for a SHA256 hash or raw PublicKey,
	// and 4 bytes of checksum.
	return base58.CheckEncode(hashokey, netID)
}

// DecodeAddress decodes the string encoding of an address and returns
// the Address and netId
func DecodeAddress(addr string) (hashokey Address, netID byte, err error) {
	if len(addr) == 50 || len(addr) == 51 {
		hashokey, netID, err = base58.CheckDecode(addr)
	}

	return
}

func AddressFromPubKey(pk *[32]byte, netID byte) string {
	hash := notaryapi.Sha(pk[:])
	return EncodeAddress(hash.Bytes, netID)
}
