package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"sync"
	//	"time"
)

const DBlockVersion = 1

type DChain struct {
	ChainID     *Hash
	Blocks      []*DBlock
	BlockMutex  sync.Mutex
	NextBlock   *DBlock
	NextBlockID uint64
	IsValidated bool
}

type DBlock struct {
	//Marshalized
	Header    *DBlockHeader
	DBEntries []*DBEntry

	//Not Marshalized
	Chain       *DChain
	IsSealed    bool
	DBHash      *Hash
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
	BlockID       uint64
	PrevBlockHash *Hash
	MerkleRoot    *Hash
	Version       int32
	//	TimeStamp     int64
	StartTime  uint64
	BatchFlag  byte // 1: start of the batch
	EntryCount uint32
}

type DBEntry struct {
	MerkleRoot *Hash // Different MR in EBlockHeader
	ChainID    *Hash

	// not marshalllized
	hash   *Hash
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

func NewDBInfoFromDBlock(b *DBlock) *DBInfo {
	e := &DBInfo{}
	e.DBHash = b.DBHash
	e.DBMerkleRoot = b.Header.MerkleRoot //?? double check

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

func (e *DBEntry) MarshalledSize() (size uint64) {
	size += e.ChainID.MarshalledSize() // Chain ID
	size += e.MerkleRoot.MarshalledSize()
	return size
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
		`BlockID`:       reflect.ValueOf(b.BlockID),
		`EntryCount`:    reflect.ValueOf(b.EntryCount),
		`MerkleRoot`:    reflect.ValueOf(b.MerkleRoot),
		`PrevBlockHash`: reflect.ValueOf(b.PrevBlockHash),
		//		`TimeStamp`: reflect.ValueOf(b.TimeStamp),
	}
	return fields
}

func (b *DBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, b.BlockID)

	data, err = b.PrevBlockHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = b.MerkleRoot.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, b.Version)
	//	binary.Write(&buf, binary.BigEndian, b.TimeStamp)
	binary.Write(&buf, binary.BigEndian, b.EntryCount)

	return buf.Bytes(), err
}

func (b *DBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 8
	size += b.PrevBlockHash.MarshalledSize()
	size += b.MerkleRoot.MarshalledSize()
	size += 4
	//	size += 8
	size += 4

	return size
}

func (b *DBlockHeader) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]

	b.PrevBlockHash = new(Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]

	b.MerkleRoot = new(Hash)
	b.MerkleRoot.UnmarshalBinary(data)
	data = data[b.MerkleRoot.MarshalledSize():]

	version, data := binary.BigEndian.Uint32(data[0:4]), data[4:]
	//	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
	b.EntryCount, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	b.Version = int32(version)
	//	b.TimeStamp = int64(timeStamp)

	return nil
}

func NewDBlockHeader(blockId uint64, prevHash *Hash, version int32,
	count uint32) *DBlockHeader {
	return &DBlockHeader{
		Version:       version,
		PrevBlockHash: prevHash,
		//		TimeStamp:     time.Now().Unix(),
		EntryCount: count,
		BlockID:    blockId,
	}
}

/*
func (b *DBlockHeader) RealTime() time.Time {
	return time.Unix(b.TimeStamp, 0)
}
*/
func CreateDBlock(chain *DChain, prev *DBlock, cap uint) (b *DBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}

	b = new(DBlock)

	var prevHash *Hash
	if prev == nil {
		prevHash = NewHash()
	} else {
		prevHash, err = CreateHash(prev)
	}

	b.Header = NewDBlockHeader(chain.NextBlockID, prevHash, DBlockVersion,
		uint32(0))
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
/*
	if len(c.NextBlock.DBEntries) < 1 {
		panic ("DBEntries not initialized properly for block: " + string(c.NextBlockID))
	}
*/
	c.BlockMutex.Lock()
	// Cblock is always at the first entry
	//c.NextBlock.DBEntries[0] = dbEntry
	c.NextBlock.DBEntries = append(c.NextBlock.DBEntries, dbEntry)	
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

	if len(c.NextBlock.DBEntries) < 2 {
		panic ("DBEntries not initialized properly for block: " + string(c.NextBlockID))
	}
	c.BlockMutex.Lock()
	// Factoid entry is alwasy at the same position
	c.NextBlock.DBEntries[1] = dbEntry
	c.BlockMutex.Unlock()

	return nil
}

// Add DBlock to the chain in memory
func (c *DChain) AddDBlockToDChain(b *DBlock) (err error) {

	// Increase the slice capacity if needed
	if b.Header.BlockID >= uint64(cap(c.Blocks)) {
		temp := make([]*DBlock, len(c.Blocks), b.Header.BlockID*2)
		copy(temp, c.Blocks)
		c.Blocks = temp
	}

	// Increase the slice length if needed
	if b.Header.BlockID >= uint64(len(c.Blocks)) {
		c.Blocks = c.Blocks[0 : b.Header.BlockID+1]
	}

	c.Blocks[b.Header.BlockID] = b

	return nil
}

func (b *DBlock) MarshalBinary() (data []byte, err error) {
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

func (b *DBlock) CalculateMerkleRoot() (mr *Hash, err error) {
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

func (b *DBlock) MarshalledSize() uint64 {
	var size uint64 = 0

	size += b.Header.MarshalledSize()
	size += 4 // len(Entries) uint32

	for _, dbEntry := range b.DBEntries {
		size += dbEntry.MarshalledSize()
	}

	return 0
}

func (b *DBlock) UnmarshalBinary(data []byte) (err error) {
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
		data = data[b.DBEntries[i].MarshalledSize():]
	}

	return nil
}

func (b *DBlock) EncodableFields() map[string]reflect.Value {
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
