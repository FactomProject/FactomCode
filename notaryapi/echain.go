package notaryapi

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	
	"encoding/binary"
	"sync"
	"strings"
	"time"

)
// Size of array used to store sha hashes.  See ShaHash.
const Separator = "/"

type EChain struct {
	//Marshalized
	ChainID 	*Hash
	Name		[][]byte
	FirstEntry *Entry	
	//Status	uint8
	
	//Not Marshalized	
	Blocks 		[]*EBlock
	CurrentBlock *EBlock	
	BlockMutex 	sync.Mutex	
	NextBlockID uint64	
}

type EBlock struct {

	//Marshalized
	Header *EBlockHeader
	EBEntries []*EBEntry

	//Not Marshalized
	EBHash *Hash 	
	MerkleRoot *Hash
	Salt *Hash
	Chain *EChain	
	IsSealed bool
}

type EBInfo struct {

    EBHash *Hash
	MerkleRoot *Hash    
    DBHash *Hash
    DBBlockNum uint64
    ChainID *Hash
    //FBOffset uint64
    //EntryInfoArray *[]EntryInfo //not marshalized in db
  
}

type EBlockHeader struct {
//	ChainID []byte // ?? put in ebinfo for now
	BlockID uint64
	PrevBlockHash *Hash
	//MerkleRoot *Hash
	TimeStamp int64
}

const eBlockHeaderLen = 80

type EBEntry struct {
	timeStamp int64 
	hash *Hash
	
	ChainID *[]byte // not marshalllized
	status int8 //for future use??

}

func NewEBEntry(h *Hash, id *[]byte) *EBEntry {
	e := &EBEntry{}
	e.StampTime()
	e.hash = h
	e.ChainID = id
	return e
}

func (e *EBEntry) Hash() *Hash {
	return e.hash
}

func (e *EBEntry) SetHash( binaryHash []byte)  {
	h := new(Hash)
	h.Bytes = binaryHash
	e.hash = h
}

func (e *EBEntry) TimeStamp() int64 {
	return e.timeStamp
}


func (e *EBEntry) GetBinaryTimeStamp() (binaryTimeStamp []byte)  {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(e.timeStamp)) 
	return b
	
}

func (e *EBEntry) SetTimeStamp(binaryTime []byte)  {
 	
 	e.timeStamp = int64(binary.BigEndian.Uint64(binaryTime))	

}

func (e *EBEntry) SetIntTimeStamp(ts int64)  {
 	
 	e.timeStamp = ts	

}


func (e *EBEntry) RealTime() time.Time {
	return time.Unix(e.timeStamp, 0)
}

func (e *EBEntry) StampTime() {
	e.timeStamp = time.Now().Unix()
}

func (e *EBEntry) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`TimeStamp`: reflect.ValueOf(e.TimeStamp()),
		`Hash`: reflect.ValueOf(e.Hash()),
	}
	return fields
}


func (e *EBEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	
	binary.Write(&buf, binary.BigEndian, e.TimeStamp())
	
	data, _ := e.Hash().MarshalBinary()
	buf.Write(data)
	

	
	return buf.Bytes(), nil
}

func (e *EBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 // TimeStamp() int64	
	size += e.Hash().MarshalledSize()

	
	return size
}

func (e *EBEntry) UnmarshalBinary(data []byte) (err error) {

	
	timeStamp,	data := binary.BigEndian.Uint64(data[:8]), data[8:]
	e.timeStamp = int64(timeStamp)
		
	e.hash = new(Hash)
	e.hash.UnmarshalBinary(data)


	
	return nil
}


func (e *EBEntry) ShaHash() *Hash {
	byteArray, _ := e.MarshalBinary()
	return Sha(byteArray)
}


func (b *EBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PrevBlockHash.MarshalBinary()
	buf.Write(data)
/*	
	data, _ = b.MerkleRoot.MarshalBinary()
	buf.Write(data)
*/		
	binary.Write(&buf, binary.BigEndian, b.TimeStamp)
	
	return buf.Bytes(), err
}

func (b *EBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 
	size += b.PrevBlockHash.MarshalledSize()
//	size += b.MerkleRoot.MarshalledSize()
	size += 8
	
	return size
}

func (b *EBlockHeader) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.PrevBlockHash = new(Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]
/*	
	b.MerkleRoot = new(Hash)
	b.MerkleRoot.UnmarshalBinary(data)
	data = data[b.MerkleRoot.MarshalledSize():]
*/	
	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
	b.TimeStamp = int64(timeStamp)

	return nil
}


func NewEBlockHeader(blockId uint64, prevHash *Hash, merkle *Hash) *EBlockHeader {

	return &EBlockHeader{
		PrevBlockHash:  prevHash,
		//MerkleRoot: merkle,
		TimeStamp:  time.Now().Unix(),
		BlockID:    blockId,
	}
}

func (e *EBlockHeader) RealTime() time.Time {
	return time.Unix(e.TimeStamp, 0)
}

func CreateBlock(chain *EChain, prev *EBlock, capacity uint) (b *EBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}
	
	b = new(EBlock)
	
	var prevHash *Hash
	if prev == nil {
		prevHash = EmptyHash()
	} else {
		prevHash, err = CreateHash(prev)
	}
	
	b.Header = NewEBlockHeader(chain.NextBlockID, prevHash, EmptyHash())
	
	b.Chain = chain
	
	b.EBEntries = make([]*EBEntry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	b.IsSealed = false
	
	return b, err
}

func (b *EBlock) AddEBEntry(e *Entry) (err error) {
	h, err := CreateHash(e)
	if err != nil { return }
	
	s, err := CreateHash(b.Salt, h)
	if err != nil { return }


 	ebEntry := NewEBEntry(h, &b.Chain.ChainID.Bytes)	
 	//ebEntry.SetIntTimeStamp(e.TimeStamp())
	
	b.EBEntries = append(b.EBEntries, ebEntry) 
	b.Salt = s
	
	return
}

func (b *EBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
/* It should not be created in MarshalBianry method
// because the header needs to be sealed before this method is called
	hashes := make([]*Hash, len(b.EBEntries))
	for i, entry := range b.EBEntries {
		data, _ := entry.MarshalBinary()
		hashes[i] = Sha(data)
		//fmt.Println("i=", i, ", hash=", hashes[i])
	}
	
	merkle := BuildMerkleTreeStore(hashes)
	b.Header.MerkleRoot = merkle[len(merkle) - 1]
*/	
	data, _ = b.Header.MarshalBinary()
	buf.Write(data)
	
	//binary.Write(&buf, binary.BigEndian, b.BlockID)
	//data, _ = b.PreviousHash.MarshalBinary()
	//buf.Write(data)
	
	count := uint64(len(b.EBEntries))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i = i + 1 {
		data, _ := b.EBEntries[i].MarshalBinary()
		buf.Write(data)
	}
	
	//data, _ = b.Salt.MarshalBinary()
	//buf.Write(data)
	
	return buf.Bytes(), err
}

func (b *EBlock) MarshalledSize() uint64 {
	var size uint64 = 0
	
	//size += 8 // BlockID uint64
	//size += b.PreviousHash.MarshalledSize()
	
	size += b.Header.MarshalledSize()
	size += 8 // len(Entries) uint64
	//size += b.Salt.MarshalledSize()
	
	for _, ebentry := range b.EBEntries {
		size += ebentry.MarshalledSize()
	}
	
	fmt.Println("block.MarshalledSize=", size)
	
	return size
}

func (b *EBlock) UnmarshalBinary(data []byte) (err error) {
	//b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	//b.PreviousHash = new(Hash)
	//b.PreviousHash.UnmarshalBinary(data)
	//data = data[b.PreviousHash.MarshalledSize():]
	
	ebh := new(EBlockHeader)
	ebh.UnmarshalBinary(data)
	b.Header = ebh
	
	data = data[ebh.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[0:8]), data[8:]
	//fmt.Println("block.entry.count=", count)
	
	b.EBEntries = make([]*EBEntry, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.EBEntries[i] = new(EBEntry)
		err = b.EBEntries[i].UnmarshalBinary(data)
		if err != nil { return }
		data = data[b.EBEntries[i].MarshalledSize():]
	}
	
	//b.Salt = new(Hash)
	//b.Salt.UnmarshalBinary(data)
	//data = data[b.Salt.MarshalledSize():]
	
	return nil
}


func (b *EBInfo) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.EBHash.MarshalBinary()
	buf.Write(data)

	data, _ = b.MerkleRoot.MarshalBinary()
	buf.Write(data)
	
	data, _ = b.DBHash.MarshalBinary()
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, b.DBBlockNum)
	
	data, _ = b.ChainID.MarshalBinary()	
	buf.Write(data) 
	
	return buf.Bytes(), err
}

func (b *EBInfo) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33	//b.EBHash
	size += 33	//b.MerkleRoot	
	size += 33  //b.FBHash
	size += 8 	//b.FBBlockNum
	size += 33 	//b.ChainID	
	
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

	data, _ = b.ChainID.MarshalBinary()
	buf.Write(data)
	
	count := len(b.Name)
	binary.Write(&buf, binary.BigEndian, uint64(count))	

	for _, bytes := range b.Name {
		count = len(bytes)
		binary.Write(&buf, binary.BigEndian, uint64(count))	
		buf.Write(bytes)
	}
	if b.FirstEntry != nil {
		data, _ = b.FirstEntry.MarshalBinary()
		buf.Write(data)
	}
	
	return buf.Bytes(), err
}

func (b *EChain) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33	//b.ChainID
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
	
	for i:=uint64(0); i<count; i++{
		length := binary.BigEndian.Uint64(data[0:8])		
		data = data[8:]		
		b.Name[i] = data[:length]
		data = data[length:]
	}
	
	if len(data) > HashSize{
		b.FirstEntry = new(Entry)
		b.FirstEntry.UnmarshalBinary(data)	
	}
	return nil
}

// For Json marshaling
func (b *EChain) EncodableFields() map[string]reflect.Value {
	fields := map[string]reflect.Value{
		`ChainID`: reflect.ValueOf(b.ChainID),
		`Name`: reflect.ValueOf(b.Name),
		`FirstEntry`: reflect.ValueOf(b.FirstEntry),		
	}
	return fields
}

// To generate a chain id (hash) from a binary array name
// The algorithm is chainID = Sha(Sha(Name[0]) + Sha(Name[1] + ... + Sha(Name[n])
func (b *EChain) GenerateIDFromName() (chainID *Hash, err error) {
	byteSlice := make ([]byte, 0, 32)
	for _, bytes := range b.Name{
		byteSlice = append(byteSlice, Sha(bytes).Bytes ...)
	}
	b.ChainID = Sha(byteSlice)
	return b.ChainID, nil
}

// To encode the binary name to a string to enable internal path search in db
// The algorithm is PathString = Hex(Name[0]) + ":" + Hex(Name[0]) + ":" + ... + Hex(Name[n])
func EncodeChainNameToString(name [][]byte) (pathstr string ) {
	pathstr = ""
	for _, bytes := range name{
		pathstr = pathstr + EncodeBinary(&bytes) + Separator
	}
	
	if len(pathstr)>0{
		pathstr = pathstr[:len(pathstr)-1]
	}
	
	return pathstr
}

// To decode the binary name to a string to enable internal path search in db
// The algorithm is PathString = Hex(Name[0]) + ":" + Hex(Name[0]) + ":" + ... + Hex(Name[n])
func DecodeStringToChainName(pathstr string) (name [][]byte) {
	strArray := strings.Split(pathstr, Separator)
	bArray := make ([][]byte, 0, 32)
	for _, str := range strArray{
		bytes,_ := DecodeBinary(&str)
		bArray = append(bArray, bytes)
	}
	return bArray
}