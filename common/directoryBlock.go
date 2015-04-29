// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

const DBlockVersion = 0

type DChain struct {
	ChainID         *Hash
	Blocks          []*DirectoryBlock
	BlockMutex      sync.Mutex
	NextBlock       *DirectoryBlock
	NextBlockHeight uint32
	IsValidated     bool
}

type DirectoryBlock struct {
	//Marshalized
	Header    *DBlockHeader
	DBEntries []*DBEntry

	//Not Marshalized
	Chain       *DChain
	IsSealed    bool
	DBHash      *Hash
	KeyMR       *Hash
	IsSavedInDB bool
}

type DBInfo struct {

	// Serial hash for the directory block
	DBHash *Hash

	// BTCTxHash is the Tx hash returned from rpcclient.SendRawTransaction
	BTCTxHash *Hash // use string or *btcwire.ShaHash ???

	// BTCTxOffset is the index of the TX in this BTC block
	BTCTxOffset int

	// BTCBlockHeight is the height of the block where this TX is stored in BTC
	BTCBlockHeight int32

	//BTCBlockHash is the hash of the block where this TX is stored in BTC
	BTCBlockHash *Hash // use string or *btcwire.ShaHash ???

	// DBMerkleRoot is the merkle root of the Directory Block
	// and is written into BTC as OP_RETURN data
	DBMerkleRoot *Hash
}

type DBlockHeader struct {
	Version       byte
	NetworkID     uint32
	BodyMR        *Hash
	PrevKeyMR     *Hash
	PrevBlockHash *Hash
	BlockHeight   uint32

	StartTime uint64 //??

	EntryCount uint32
}

type DBEntry struct {
	MerkleRoot *Hash // Different MR in EBlockHeader
	ChainID    *Hash

	// not marshalllized
	hash   *Hash  // to be removed??
	status int8 //for future use??
}

func NewDBEntry(eb *EBlock) *DBEntry {
	e := &DBEntry{}
	e.hash = eb.EBHash

	e.ChainID = eb.Chain.ChainID
	e.MerkleRoot = eb.MerkleRoot

	return e
}

func NewDBEntryFromCBlock(cb *CBlock) *DBEntry {
	e := &DBEntry{}
	e.hash = cb.CBHash

	e.ChainID = cb.Chain.ChainID
	e.MerkleRoot = cb.CBHash //To use MerkleRoot??

	return e
}

func NewDBEntryFromABlock(b *AdminBlock) *DBEntry {
	e := &DBEntry{}
	e.hash = b.ABHash

	e.ChainID = b.Header.ChainID
	e.MerkleRoot = b.ABHash

	return e
}

func NewDBInfoFromDBlock(b *DirectoryBlock) *DBInfo {
	e := &DBInfo{}
	e.DBHash = b.DBHash
	e.DBMerkleRoot = b.Header.BodyMR //?? double check

	return e
}

func (e *DBEntry) Hash() *Hash {
	return e.hash
}

func (e *DBEntry) SetHash(binaryHash []byte) {
	h := new(Hash)
	h.Bytes = binaryHash
	e.hash = h
}

func (e *DBEntry) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`MerkleRoot`: reflect.ValueOf(e.MerkleRoot),
		`ChainID`:    reflect.ValueOf(e.ChainID),
	}
	return fields
}

func (e *DBEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = e.ChainID.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = e.MerkleRoot.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

func (e *DBEntry) UnmarshalBinary(data []byte) (err error) {
	e.ChainID = new(Hash)
	err = e.ChainID.UnmarshalBinary(data[:33])
	if err != nil {
		return
	}

	e.MerkleRoot = new(Hash)
	err = e.MerkleRoot.UnmarshalBinary(data[33:])
	if err != nil {
		return
	}

	return nil
}

func (e *DBEntry) ShaHash() *Hash {
	byteArray, _ := e.MarshalBinary()
	return Sha(byteArray)
}

func (b *DBlockHeader) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`BlockHeight`:   reflect.ValueOf(b.BlockHeight),
		`EntryCount`:    reflect.ValueOf(b.EntryCount),
		`BodyMR`:        reflect.ValueOf(b.BodyMR),
		`PrevBlockHash`: reflect.ValueOf(b.PrevBlockHash),
	}
	return fields
}

func (b *DBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{b.Version})
	binary.Write(&buf, binary.BigEndian, b.NetworkID)

	data, err = b.BodyMR.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = b.PrevKeyMR.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = b.PrevBlockHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, b.BlockHeight)

	binary.Write(&buf, binary.BigEndian, b.StartTime)

	binary.Write(&buf, binary.BigEndian, b.EntryCount)

	return buf.Bytes(), err
}

func (b *DBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 1
	size += 4
	size += b.BodyMR.MarshalledSize()
	size += b.PrevKeyMR.MarshalledSize()
	size += b.PrevBlockHash.MarshalledSize()
	size += 4 //db height
	size += 8 //start time
	size += 4

	return size
}

func (b *DBlockHeader) UnmarshalBinary(data []byte) (err error) {

	b.Version, data = data[0], data[1:]

	b.NetworkID, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	b.BodyMR = new(Hash)
	b.BodyMR.UnmarshalBinary(data)
	data = data[b.BodyMR.MarshalledSize():]

	b.PrevKeyMR = new(Hash)
	b.PrevKeyMR.UnmarshalBinary(data)
	data = data[b.PrevKeyMR.MarshalledSize():]

	b.PrevBlockHash = new(Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]

	b.BlockHeight, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	b.StartTime, data = binary.BigEndian.Uint64(data[0:8]), data[8:]

	b.EntryCount, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	return nil
}

func CreateDBlock(chain *DChain, prev *DirectoryBlock, cap uint) (b *DirectoryBlock, err error) {
	if prev == nil && chain.NextBlockHeight != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockHeight == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}

	b = new(DirectoryBlock)

	b.Header = new(DBlockHeader)
	b.Header.Version = VERSION_0

	if prev == nil {
		b.Header.PrevBlockHash = NewHash()
		b.Header.PrevKeyMR = NewHash()
	} else {
		b.Header.PrevBlockHash, err = CreateHash(prev)
		if prev.KeyMR == nil {
			prev.BuildKeyMerkleRoot()
		}
		b.Header.PrevKeyMR = prev.KeyMR
	}

	b.Header.BlockHeight = chain.NextBlockHeight
	b.Chain = chain
	b.DBEntries = make([]*DBEntry, 0, cap)
	b.IsSealed = false

	return b, err
}

// Add DBEntry from an Entry Block
func (c *DChain) AddEBlockToDBEntry(eb *EBlock) (err error) {

	dbEntry := NewDBEntry(eb)

	c.BlockMutex.Lock()
	c.NextBlock.DBEntries = append(c.NextBlock.DBEntries, dbEntry)
	c.BlockMutex.Unlock()

	return nil
}

// Add DBEntry from an Entry Credit Block
func (c *DChain) AddCBlockToDBEntry(cb *CBlock) (err error) {

	dbEntry := NewDBEntryFromCBlock(cb)

	if len(c.NextBlock.DBEntries) < 3 {
		panic("DBEntries not initialized properly for block: " + string(c.NextBlockHeight))
	}

	c.BlockMutex.Lock()
	// Cblock is always at the first entry
	c.NextBlock.DBEntries[1] = dbEntry // First three entries are ABlock, CBlock, FBlock
	c.BlockMutex.Unlock()

	return nil
}

// Add DBEntry from an Admin Block
func (c *DChain) AddABlockToDBEntry(b *AdminBlock) (err error) {

	dbEntry := &DBEntry{}
	dbEntry.ChainID = b.Header.ChainID
	dbEntry.MerkleRoot = b.ABHash

	if len(c.NextBlock.DBEntries) < 3 {
		panic("DBEntries not initialized properly for block: " + string(c.NextBlockHeight))
	}

	c.BlockMutex.Lock()
	// Ablock is always at the first entry
	// First three entries are ABlock, CBlock, FBlock		
	c.NextBlock.DBEntries[0] = dbEntry 
	c.BlockMutex.Unlock()

	return nil
}

// Add DBEntry
func (c *DChain) AddDBEntry(dbEntry *DBEntry) (err error) {

	c.BlockMutex.Lock()
	c.NextBlock.DBEntries = append(c.NextBlock.DBEntries, dbEntry)
	c.BlockMutex.Unlock()

	return nil
}

// Add DBEntry from a Factoid Block
func (c *DChain) AddFBlockMRToDBEntry(dbEntry *DBEntry) (err error) {

	fmt.Println("AddFDBlock >>>>>")

	if len(c.NextBlock.DBEntries) < 3 {
		panic("DBEntries not initialized properly for block: " + string(c.NextBlockHeight))
	}
	c.BlockMutex.Lock()
	// Factoid entry is alwasy at the same position
	// First three entries are ABlock, CBlock, FBlock	
	c.NextBlock.DBEntries[2] = dbEntry 
	c.BlockMutex.Unlock()

	return nil
}

// Add DBlock to the chain in memory
func (c *DChain) AddDBlockToDChain(b *DirectoryBlock) (err error) {

	// Increase the slice capacity if needed
	if b.Header.BlockHeight >= uint32(cap(c.Blocks)) {
		temp := make([]*DirectoryBlock, len(c.Blocks), b.Header.BlockHeight*2)
		copy(temp, c.Blocks)
		c.Blocks = temp
	}

	// Increase the slice length if needed
	if b.Header.BlockHeight >= uint32(len(c.Blocks)) {
		c.Blocks = c.Blocks[0 : b.Header.BlockHeight+1]
	}

	c.Blocks[b.Header.BlockHeight] = b

	return nil
}

func (b *DirectoryBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = b.Header.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	count := uint32(len(b.DBEntries))
	// need to get rid of count, duplicated with blockheader.entrycount
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint32(0); i < count; i = i + 1 {
		data, err = b.DBEntries[i].MarshalBinary()
		if err != nil {
			return
		}
		buf.Write(data)
	}

	return buf.Bytes(), err
}

func (b *DirectoryBlock) BuildBodyMR() (mr *Hash, err error) {
	hashes := make([]*Hash, len(b.DBEntries))
	for i, entry := range b.DBEntries {
		data, _ := entry.MarshalBinary()
		hashes[i] = Sha(data)
	}
	// Verify bodyMR here??

	if len(hashes) == 0 {
		hashes = append(hashes, Sha(nil))
	}

	merkle := BuildMerkleTreeStore(hashes)
	return merkle[len(merkle)-1], nil
}

func (b *DirectoryBlock) BuildKeyMerkleRoot() (err error) {

	// Create the Entry Block Key Merkle Root from the hash of Header and the Body Merkle Root
	hashes := make([]*Hash, 0, 2)
	binaryEBHeader, _ := b.Header.MarshalBinary()
	hashes = append(hashes, Sha(binaryEBHeader))
	hashes = append(hashes, b.Header.BodyMR)
	merkle := BuildMerkleTreeStore(hashes)
	b.KeyMR = merkle[len(merkle)-1] // MerkleRoot is not marshalized in Dir Block

	return
}

func (b *DirectoryBlock) UnmarshalBinary(data []byte) (err error) {
	fbh := new(DBlockHeader)
	fbh.UnmarshalBinary(data)
	b.Header = fbh
	data = data[fbh.MarshalledSize():]

	count, data := binary.BigEndian.Uint32(data[0:4]), data[4:]
	b.DBEntries = make([]*DBEntry, count)
	for i := uint32(0); i < count; i++ {
		b.DBEntries[i] = new(DBEntry)
		err = b.DBEntries[i].UnmarshalBinary(data)
		if err != nil {
			return
		}
		data = data[HASH_LENGTH*2:]
	}

	return nil
}

func (b *DirectoryBlock) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`Header`:    reflect.ValueOf(b.Header),
		`DBEntries`: reflect.ValueOf(b.DBEntries),
		`DBHash`:    reflect.ValueOf(b.DBHash),
	}
	return fields
}

func (b *DBInfo) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = b.DBHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = b.BTCTxHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, b.BTCTxOffset)
	binary.Write(&buf, binary.BigEndian, b.BTCBlockHeight)

	data, err = b.BTCBlockHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = b.DBMerkleRoot.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	return buf.Bytes(), err
}

func (b *DBInfo) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33 //DBHash
	size += 33 //BTCTxHash
	size += 4  //BTCTxOffset
	size += 4  //BTCBlockHeight
	size += 33 //BTCBlockHash
	size += 33 //DBMerkleRoot

	return size
}

func (b *DBInfo) UnmarshalBinary(data []byte) (err error) {
	b.DBHash = new(Hash)
	b.DBHash.UnmarshalBinary(data[:33])

	b.BTCTxHash = new(Hash)
	b.BTCTxHash.UnmarshalBinary(data[:33])
	data = data[33:]

	b.BTCTxOffset = int(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]

	b.BTCBlockHeight = int32(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]

	b.BTCBlockHash = new(Hash)
	b.BTCBlockHash.UnmarshalBinary(data[:33])

	b.DBMerkleRoot = new(Hash)
	b.DBMerkleRoot.UnmarshalBinary(data[:33])

	return nil
}
