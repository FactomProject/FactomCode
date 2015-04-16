package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
//	"time"
)

const (
	eBlockHeaderLen = 80
	Separator       = "/"
)

type EChain struct {
	//Marshalized
	ChainID    *Hash
	Name       [][]byte
	FirstEntry *Entry

	//Not Marshalized
	//Blocks       []*EBlock
	NextBlock *EBlock
	NextBlockID  uint64
	BlockMutex   sync.Mutex	
}

type EBlock struct {
	//Marshalized
	Header    *EBlockHeader
	EBEntries []*EBEntry

	//Not Marshalized
	EBHash     *Hash
	MerkleRoot *Hash
	Chain      *EChain
	IsSealed   bool
}

type EBInfo struct {
	EBHash     *Hash
	MerkleRoot *Hash
	DBHash     *Hash
	DBBlockNum uint64
	ChainID    *Hash
}

type EBlockHeader struct {
	Version 	  byte
	NetworkID	  uint32
	ChainID		  *Hash
	BodyMR		  *Hash
	PrevKeyMR	  *Hash
	PrevHash      *Hash	
	EBHeight       uint64
	DBHeight       uint64
	StartTime      uint64
	EntryCount	   uint32	
}

type EBEntry struct {
	EntryHash      *Hash
	
	//ChainID   *[]byte // not marshalllized
}

func NewEBEntry(h *Hash) *EBEntry {
	return &EBEntry{
		EntryHash: h,}
}

func (e *EBEntry) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
//		`TimeStamp`: reflect.ValueOf(e.TimeStamp()),
		`EntryHash`:      reflect.ValueOf(e.EntryHash),
	}
	return fields
}

func (e *EBEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

//	binary.Write(&buf, binary.BigEndian, e.TimeStamp())

	data, err := e.EntryHash.MarshalBinary()
	if err != nil {
		return nil, err
	}

	buf.Write(data)

	return buf.Bytes(), nil
}

func (e *EBEntry) MarshalledSize() uint64 {
	var size uint64 = 0

	size += e.EntryHash.MarshalledSize()
				
	return size
}

func (e *EBEntry) UnmarshalBinary(data []byte) (err error) {
//	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
//	e.timeStamp = int64(timeStamp)
	e.EntryHash = new(Hash)
	e.EntryHash.UnmarshalBinary(data)
	return nil
}

func (e *EBEntry) ShaHash() *Hash {
	byteArray, _ := e.MarshalBinary()
	return Sha(byteArray)
}

func (b *EBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{b.Version})
	binary.Write(&buf, binary.BigEndian, b.NetworkID)	
	data, err = b.ChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)	

	data, err = b.BodyMR.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)	

	data, err = b.PrevKeyMR.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)	
	
	data, err = b.PrevHash.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)		
	
	binary.Write(&buf, binary.BigEndian, b.EBHeight)

	binary.Write(&buf, binary.BigEndian, b.DBHeight)
	
	binary.Write(&buf, binary.BigEndian, b.StartTime)	

	binary.Write(&buf, binary.BigEndian, b.EntryCount)
	
	return buf.Bytes(), err
}

func (b *EBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0

	size += 1	
	size += 4
	size += b.ChainID.MarshalledSize()
	size += b.BodyMR.MarshalledSize()
	size += b.PrevKeyMR.MarshalledSize()
	size += b.PrevHash.MarshalledSize()
	size += 8	
	size += 8	
	size += 8	
	size += 4	

	return size
}

func (b *EBlockHeader) UnmarshalBinary(data []byte) (err error) {
	
	b.Version, data = data[0], data[1:]
	
	b.NetworkID, data = binary.BigEndian.Uint32(data[0:4]), data[4:]

	b.ChainID = new(Hash)
	b.ChainID.UnmarshalBinary(data)
	data = data[b.ChainID.MarshalledSize():]

	b.BodyMR = new(Hash)
	b.BodyMR.UnmarshalBinary(data)
	data = data[b.BodyMR.MarshalledSize():]

	b.PrevKeyMR = new(Hash)
	b.PrevKeyMR.UnmarshalBinary(data)
	data = data[b.PrevKeyMR.MarshalledSize():]

	b.PrevHash = new(Hash)
	b.PrevHash.UnmarshalBinary(data)
	data = data[b.PrevHash.MarshalledSize():]

	b.EBHeight, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.DBHeight, data = binary.BigEndian.Uint64(data[0:8]), data[8:]	

	b.StartTime, data = binary.BigEndian.Uint64(data[0:8]), data[8:]	
	
	b.EntryCount, data = binary.BigEndian.Uint32(data[0:4]), data[4:]	
			
	return nil
}


func CreateBlock(chain *EChain, prev *EBlock, capacity uint) (b *EBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}

	b = new(EBlock)

	b.Header = new (EBlockHeader)
	b.Header.Version = VERSION_0
	b.Header.ChainID = chain.ChainID
	b.Header.EBHeight = chain.NextBlockID
	if prev == nil {
		b.Header.PrevHash = NewHash()
		b.Header.PrevKeyMR = NewHash()		
	} else {
		b.Header.PrevHash = prev.EBHash
		if prev.MerkleRoot == nil {
			prev.BuildMerkleRoot()
		}
		b.Header.PrevKeyMR = prev.MerkleRoot		
	}


	b.Chain = chain

	b.EBEntries = make([]*EBEntry, 0, capacity)

	b.IsSealed = false

	return b, err
}

func (b *EBlock) AddEBEntry(e *Entry) (err error) {
	h, err := CreateHash(e)
	if err != nil {
		return
	}

	ebEntry := NewEBEntry(h)

	b.EBEntries = append(b.EBEntries, ebEntry)

	return
}

func (b *EBlock) AddEndOfMinuteMarker(eomType byte) (err error) {
	bytes:= make([]byte, 32)
	bytes[31] = eomType
	
	h, _ := NewShaHash(bytes)

	ebEntry := NewEBEntry(h)

	b.EBEntries = append(b.EBEntries, ebEntry)

	return
}

func (block *EBlock) BuildMerkleRoot() (err error) {
	// Create the Entry Block Boday Merkle Root from EB Entries
	hashes := make([]*Hash, 0, len(block.EBEntries))
	for _, entry := range block.EBEntries {
		hashes = append(hashes, entry.EntryHash)
	}
	merkle := BuildMerkleTreeStore(hashes)	
		
	// Create the Entry Block Key Merkle Root from the hash of Header and the Body Merkle Root	
	hashes = make([]*Hash, 0, 2)	
	binaryEBHeader, _ := block.Header.MarshalBinary()
	hashes = append(hashes, Sha(binaryEBHeader))
	hashes = append(hashes, block.Header.BodyMR)	
	merkle = BuildMerkleTreeStore(hashes)
	block.MerkleRoot = merkle[len(merkle)-1] // MerkleRoot is not marshalized in Entry Block

	return
}

func (e *EBlock) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`Header`:    reflect.ValueOf(e.Header),
		`EBEntries`: reflect.ValueOf(e.EBEntries),
	}
	return fields
}

func (b *EBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = b.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	//binary.Write(&buf, binary.BigEndian, b.BlockID)
	//data, _ = b.PreviousHash.MarshalBinary()
	//buf.Write(data)

	count := uint64(len(b.EBEntries))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i = i + 1 {
		data, err := b.EBEntries[i].MarshalBinary()
		if err != nil {
			return nil, err
		}

		buf.Write(data)
	}

	return buf.Bytes(), err
}

func (b *EBlock) MarshalledSize() (size uint64) {
	size += b.Header.MarshalledSize()
	size += 8 // len(Entries) uint64

	for _, ebentry := range b.EBEntries {
		size += ebentry.MarshalledSize()
	}

	fmt.Println("block.MarshalledSize=", size)

	return size
}

func (b *EBlock) UnmarshalBinary(data []byte) (err error) {
	ebh := new(EBlockHeader)
	ebh.UnmarshalBinary(data)
	b.Header = ebh

	data = data[ebh.MarshalledSize():]

	count, data := binary.BigEndian.Uint64(data[0:8]), data[8:]

	b.EBEntries = make([]*EBEntry, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.EBEntries[i] = new(EBEntry)
		err = b.EBEntries[i].UnmarshalBinary(data)
		if err != nil {
			return
		}
		data = data[b.EBEntries[i].MarshalledSize():]
	}

	return nil
}

func (b *EBInfo) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = b.EBHash.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	data, err = b.MerkleRoot.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	data, err = b.DBHash.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, b.DBBlockNum)

	data, err = b.ChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	return buf.Bytes(), err
}

func (b *EBInfo) MarshalledSize() (size uint64) {
	size += 33 //b.EBHash
	size += 33 //b.MerkleRoot
	size += 33 //b.FBHash
	size += 8  //b.FBBlockNum
	size += 33 //b.ChainID

	return size
}

func (b *EBInfo) UnmarshalBinary(data []byte) (err error) {
	b.EBHash = new(Hash)
	b.EBHash.UnmarshalBinary(data[:33])

	data = data[33:]
	b.MerkleRoot = new(Hash)
	b.MerkleRoot.UnmarshalBinary(data[:33])

	data = data[33:]
	b.DBHash = new(Hash)
	b.DBHash.UnmarshalBinary(data[:33])

	data = data[33:]
	b.DBBlockNum = binary.BigEndian.Uint64(data[0:8])

	data = data[8:]
	b.ChainID = new(Hash)
	b.ChainID.UnmarshalBinary(data[:33])

	return nil
}

func (b *EChain) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, err = b.ChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	count := len(b.Name)
	binary.Write(&buf, binary.BigEndian, uint64(count))

	for _, bytes := range b.Name {
		count = len(bytes)
		binary.Write(&buf, binary.BigEndian, uint64(count))
		buf.Write(bytes)
	}
	if b.FirstEntry != nil {
		data, err = b.FirstEntry.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	return buf.Bytes(), err
}

func (b *EChain) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33 //b.ChainID
	size += 8  // Name length

	for _, bytes := range b.Name {
		size += 8
		size += uint64(len(bytes))
	}

	if b.FirstEntry != nil {
		size += b.FirstEntry.MarshalledSize()
	}

	return size
}

func (b *EChain) UnmarshalBinary(data []byte) (err error) {
	b.ChainID = new(Hash)
	b.ChainID.UnmarshalBinary(data[:33])

	data = data[33:]
	count := binary.BigEndian.Uint64(data[0:8])
	data = data[8:]

	b.Name = make([][]byte, count, count)

	for i := uint64(0); i < count; i++ {
		length := binary.BigEndian.Uint64(data[0:8])
		data = data[8:]
		b.Name[i] = data[:length]
		data = data[length:]
	}

	if len(data) > HashSize {
		b.FirstEntry = new(Entry)
		b.FirstEntry.UnmarshalBinary(data)
	}
	return nil
}

// For Json marshaling
func (b *EChain) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`ChainID`:    reflect.ValueOf(b.ChainID),
		`Name`:       reflect.ValueOf(b.Name),
		`FirstEntry`: reflect.ValueOf(b.FirstEntry),
	}
	return fields
}

// To generate a chain id (hash) from a binary array name
// The algorithm is chainID = Sha(Sha(Name[0]) + Sha(Name[1] + ... + Sha(Name[n])
func (b *EChain) GenerateIDFromName() (chainID *Hash, err error) {
	byteSlice := make([]byte, 0, 32)
	for _, bytes := range b.Name {
		byteSlice = append(byteSlice, Sha(bytes).Bytes...)
	}
	b.ChainID = Sha(byteSlice)
	return b.ChainID, nil
}

// To encode the binary name to a string to enable internal path search in db
// The algorithm is PathString = Hex(Name[0]) + ":" + Hex(Name[0]) + ":" + ... + Hex(Name[n])
func EncodeChainNameToString(name [][]byte) (pathstr string) {
	for _, bytes := range name {
		pathstr = pathstr + EncodeBinary(&bytes) + Separator
	}

	if len(pathstr) > 0 {
		pathstr = pathstr[:len(pathstr)-1]
	}

	return pathstr
}

// To decode the binary name to a string to enable internal path search in db
// The algorithm is PathString = Hex(Name[0]) + ":" + Hex(Name[0]) + ":" + ... + Hex(Name[n])
func DecodeStringToChainName(pathstr string) (name [][]byte) {
	strArray := strings.Split(pathstr, Separator)
	bArray := make([][]byte, 0, 32)
	for _, str := range strArray {
		bytes, _ := DecodeBinary(&str)
		bArray = append(bArray, bytes)
	}
	return bArray
}
