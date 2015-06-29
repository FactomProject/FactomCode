// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	//"fmt"
	"sync"
)

// Administrative Chain
type AdminChain struct {
	ChainID *Hash
	Name    [][]byte

	NextBlock       *AdminBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

// Administrative Block
// This is a special block which accompanies this Directory Block.
// It contains the signatures and organizational data needed to validate previous and future Directory Blocks.
// This block is included in the DB body. It appears there with a pair of the Admin ChainID:SHA256 of the block.
// For more details, please go to:
// https://github.com/FactomProject/FactomDocs/blob/master/factomDataStructureDetails.md#administrative-block
type AdminBlock struct {
	//Marshalized
	Header    *ABlockHeader
	ABEntries []ABEntry //Interface

	//Not Marshalized
	ABHash *Hash
}

// Create an empty Admin Block
func CreateAdminBlock(chain *AdminChain, prev *AdminBlock, cap uint) (b *AdminBlock, err error) {
	if prev == nil && chain.NextBlockHeight != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockHeight == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}

	b = new(AdminBlock)

	b.Header = new(ABlockHeader)
	b.Header.ChainID = chain.ChainID

	if prev == nil {
		b.Header.PrevHash = NewHash()
	} else {

		if prev.ABHash == nil {
			prev.BuildABHash()
		}
		b.Header.PrevHash = prev.ABHash
	}

	b.Header.DBHeight = chain.NextBlockHeight
	b.ABEntries = make([]ABEntry, 0, cap)

	return b, err
}

// Build the sha hash for the admin block
func (b *AdminBlock) BuildABHash() (err error) {

	binaryAB, _ := b.MarshalBinary()
	b.ABHash = Sha(binaryAB)

	return
}

// Add an Admin Block entry to the block
func (b *AdminBlock) AddABEntry(e ABEntry) (err error) {
	b.ABEntries = append(b.ABEntries, e)
	return
}

// Add the end-of-minute marker into the admin block
func (b *AdminBlock) AddEndOfMinuteMarker(eomType byte) (err error) {
	eOMEntry := &EndOfMinuteEntry{
		entryType: TYPE_MINUTE_NUM,
		EOM_Type:  eomType}

	b.AddABEntry(eOMEntry)

	return
}

// Write out the AdminBlock to binary.
func (b *AdminBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.Header.MarshalBinary()
	buf.Write(data)

	for i := uint32(0); i < b.Header.EntryCount; i++ {
		data, _ := b.ABEntries[i].MarshalBinary()
		buf.Write(data)
	}
	return buf.Bytes(), err
}

// Admin Block size
func (b *AdminBlock) MarshalledSize() uint64 {
	var size uint64 = 0

	size += b.Header.MarshalledSize()

	for _, entry := range b.ABEntries {
		size += entry.MarshalledSize()
	}

	return size
}

// Read in the binary into the Admin block.
func (b *AdminBlock) UnmarshalBinary(data []byte) (err error) {
	h := new(ABlockHeader)
	h.UnmarshalBinary(data)
	b.Header = h

	data = data[h.MarshalledSize():]
	b.ABEntries = make([]ABEntry, b.Header.EntryCount)
	for i := uint32(0); i < b.Header.EntryCount; i++ {
		if data[0] == TYPE_DB_SIGNATURE {
			b.ABEntries[i] = new(DBSignatureEntry)
		} else if data[0] == TYPE_MINUTE_NUM {
			b.ABEntries[i] = new(EndOfMinuteEntry)
		}
		err = b.ABEntries[i].UnmarshalBinary(data)
		if err != nil {
			return
		}
		data = data[b.ABEntries[i].MarshalledSize():]
	}

	return nil
}

// Read in the binary into the Admin block.
func (b *AdminBlock) GetDBSignature() ABEntry {

	for i := uint32(0); i < b.Header.EntryCount; i++ {
		if b.ABEntries[i].Type() == TYPE_DB_SIGNATURE {
			return b.ABEntries[i]
		}
	}

	return nil
}

// Admin Block Header
type ABlockHeader struct {
	ChainID    *Hash
	PrevHash   *Hash
	DBHeight   uint32
	EntryCount uint32
	BodySize   uint32
}

// Write out the ABlockHeader to binary.
func (b *ABlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = b.ChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	data, err = b.PrevHash.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, b.DBHeight)

	binary.Write(&buf, binary.BigEndian, b.EntryCount)

	binary.Write(&buf, binary.BigEndian, b.BodySize)

	return buf.Bytes(), err
}

func (b *ABlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0

	size += uint64(HASH_LENGTH)
	size += uint64(HASH_LENGTH)
	size += 4 // DB Height
	size += 4 // Entry count
	size += 4 // Body Size

	return size
}

// Read in the binary into the ABlockHeader.
func (b *ABlockHeader) UnmarshalBinary(data []byte) (err error) {

	b.ChainID = new(Hash)
	b.ChainID.UnmarshalBinary(data)
	data = data[HASH_LENGTH:]

	b.PrevHash = new(Hash)
	b.PrevHash.UnmarshalBinary(data)
	data = data[HASH_LENGTH:]

	b.DBHeight, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	b.EntryCount, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	b.BodySize, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	return nil
}

// Generic admin block entry type
type ABEntry interface {
	Type() byte
	MarshalBinary() ([]byte, error)
	MarshalledSize() uint64
	UnmarshalBinary(data []byte) (err error)
}

// DB Signature Entry -------------------------
type DBSignatureEntry struct {
	ABEntry         //interface
	entryType       byte
	IdentityChainID *Hash
	PubKey          PublicKey
	PrevDBSig       *[64]byte
}

// Create a new DB Signature Entry
func NewDBSignatureEntry(identityChainID *Hash, sig Signature) (e *DBSignatureEntry) {
	e = new(DBSignatureEntry)
	e.entryType = TYPE_DB_SIGNATURE
	e.IdentityChainID = identityChainID
	e.PubKey = sig.Pub
	e.PrevDBSig = sig.Sig
	return
}

func (e *DBSignatureEntry) Type() byte {
	return e.entryType
}

func (e *DBSignatureEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{e.entryType})

	data, err = e.IdentityChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	_, err = buf.Write(e.PubKey.Key[:])
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(e.PrevDBSig[:])
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *DBSignatureEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 1 // Type (byte)
	size += uint64(HASH_LENGTH)
	size += uint64(HASH_LENGTH)
	size += uint64(SIG_LENGTH)

	return size
}

func (e *DBSignatureEntry) UnmarshalBinary(data []byte) (err error) {
	e.entryType, data = data[0], data[1:]

	e.IdentityChainID = new(Hash)
	e.IdentityChainID.UnmarshalBinary(data)
	data = data[HASH_LENGTH:]

	e.PubKey.Key = new([HASH_LENGTH]byte)
	copy(e.PubKey.Key[:], data[:HASH_LENGTH])
	data = data[HASH_LENGTH:]

	e.PrevDBSig = new([SIG_LENGTH]byte)
	copy(e.PrevDBSig[:], data[:SIG_LENGTH])

	return nil
}

type EndOfMinuteEntry struct {
	entryType byte
	EOM_Type  byte
}

func (m *EndOfMinuteEntry) Type() byte {
	return m.entryType
}

func (e *EndOfMinuteEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{e.entryType})

	buf.Write([]byte{e.EOM_Type})

	return buf.Bytes(), nil
}

func (e *EndOfMinuteEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 1 // Type (byte)
	size += 1 // EOM_Type (byte)

	return size
}

func (e *EndOfMinuteEntry) UnmarshalBinary(data []byte) (err error) {
	e.entryType, data = data[0], data[1:]
	e.EOM_Type, data = data[0], data[1:]

	return nil
}
