package notaryapi

import (
	"bytes"
	"errors"
	"fmt"
	"time"
	
	"encoding/binary"
	"sync"
//	"strings"

)

type CChain struct {
	ChainID 	*Hash
	Name		[][]byte
	//Status	uint8
	
	Blocks 		[]*CBlock
	BlockMutex 	sync.Mutex	
	NextBlockID uint64	
	//FirstEntry *Entry
}

type CBlock struct {

	//Marshalized
	Header *CBlockHeader
	CBEntries []*CBEntry

	//Not Marshalized
	CBHash *Hash 	
	//MerkleRoot *Hash
	Salt *Hash
	Chain *CChain	
	IsSealed bool
}



type CBInfo struct {

    CBHash *Hash 
    FBHash *Hash
    FBBlockNum uint64
    ChainID *Hash
    
}


func CreateCBlock(chain *CChain, prev *CBlock, capacity uint) (b *CBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}
	
	b = new(CBlock)
	
	var prevHash *Hash
	if prev == nil {
		prevHash = EmptyHash()
	} else {
		prevHash, err = CreateHash(prev)
	}
	
	b.Header = NewCBlockHeader(chain.NextBlockID, prevHash, EmptyHash())
	
	b.Chain = chain
	
	b.CBEntries = make([]*CBEntry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	b.IsSealed = false
	
	return b, err
}

func (b *CBlock) AddCBEntry(e *CBEntry) (err error) {

	b.CBEntries = append(b.CBEntries, e) 
	
	return
}

func (b *CBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.Header.MarshalBinary()
	buf.Write(data)
	
	count := uint64(len(b.CBEntries))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i = i + 1 {
		data, _ := b.CBEntries[i].MarshalBinary()
		buf.Write(data)
	}
	
	return buf.Bytes(), err
}

func (b *CBlock) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += b.Header.MarshalledSize()
	size += 8 // len(Entries) uint64
	
	for _, entry := range b.CBEntries {
		size += entry.MarshalledSize()
	}
	
	fmt.Println("cblock.MarshalledSize=", size)
	
	return size
}

func (b *CBlock) UnmarshalBinary(data []byte) (err error) {
	
	h := new(CBlockHeader)
	h.UnmarshalBinary(data)
	b.Header = h
	
	data = data[h.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.CBEntries = make([]*CBEntry, count)
	for i := uint64(0); i < count; i = i + 1 {
		b.CBEntries[i] = new(CBEntry)
		err = b.CBEntries[i].UnmarshalBinary(data)
		if err != nil { return }
		data = data[b.CBEntries[i].MarshalledSize():]
	}
	
	//b.Salt = new(Hash)
	//b.Salt.UnmarshalBinary(data)
	//data = data[b.Salt.MarshalledSize():]
	
	return nil
}


func (b *CBInfo) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.CBHash.MarshalBinary()
	buf.Write(data)
	
	data, _ = b.FBHash.MarshalBinary()
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, b.FBBlockNum)
	
	data, _ = b.ChainID.MarshalBinary()	
	buf.Write(data) 
	
	return buf.Bytes(), err
}

func (b *CBInfo) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33	//b.EBHash
	size += 33  //b.FBHash
	size += 8 	//b.FBBlockNum
	size += 33 	//b.ChainID	
	
	return size
}

func (b *CBInfo) UnmarshalBinary(data []byte) (err error) {
	b.CBHash = new(Hash)
	b.CBHash.UnmarshalBinary(data[:33])

	data = data[33:]
	b.FBHash = new(Hash)
	b.FBHash.UnmarshalBinary(data[:33])	
	
	data = data[33:]
	b.FBBlockNum = binary.BigEndian.Uint64(data[0:8])
	
	data = data[8:]
	b.ChainID = new(Hash)
	b.ChainID.UnmarshalBinary(data[:33])	
	
	return nil
}

func (b *CChain) MarshalBinary() (data []byte, err error) {
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

	return buf.Bytes(), err
}

func (b *CChain) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33	//b.ChainID
	size += 8  // Name length
	for _, bytes := range b.Name {
		size += 8
		size += uint64(len(bytes))
	}
	
	return size
}


func (b *CChain) UnmarshalBinary(data []byte) (err error) {
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
	
	return nil
}


//-----------------------
type CBlockHeader struct {

	BlockID uint64
	PrevBlockHash *Hash
	TimeStamp int64
	EntryCount uint32	
}

func (b *CBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	binary.Write(&buf, binary.BigEndian, b.BlockID)
	
	data, _ = b.PrevBlockHash.MarshalBinary()
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, b.TimeStamp)
	
	binary.Write(&buf, binary.BigEndian, b.EntryCount)	
	
	return buf.Bytes(), err
}

func (b *CBlockHeader) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += 8 
	size += b.PrevBlockHash.MarshalledSize()
	size += 8
	size += 4
	
	return size
}

func (b *CBlockHeader) UnmarshalBinary(data []byte) (err error) {
	b.BlockID, data = binary.BigEndian.Uint64(data[0:8]), data[8:]
	
	b.PrevBlockHash = new(Hash)
	b.PrevBlockHash.UnmarshalBinary(data)
	data = data[b.PrevBlockHash.MarshalledSize():]

	timeStamp, data := binary.BigEndian.Uint64(data[:8]), data[8:]
	b.TimeStamp = int64(timeStamp)


	b.EntryCount = binary.BigEndian.Uint32(data[:4])
	
	return nil
}


func NewCBlockHeader(blockId uint64, prevHash *Hash, merkle *Hash) *CBlockHeader {

	return &CBlockHeader{
		PrevBlockHash:  prevHash,
		TimeStamp:  time.Now().Unix(),
		BlockID:    blockId,
	}
}

//-----------------------------------------------------------

const (
	TYPE_BUY     uint8 = iota
	TYPE_PAY
)

type CBEntry struct {
	PublicKey *Hash 
	Type byte
	EntryHash * Hash
	FactomTxHash *Hash
	Credits int

}

func NewPayCBEntry(pubKey *Hash, entryHash *Hash, credits int) *CBEntry {
	e := &CBEntry{}
	e.PublicKey = pubKey
	e.Type = TYPE_PAY	
	e.EntryHash = entryHash
	e.Credits = credits
	
	return e
}

func NewBuyCBEntry(pubKey *Hash, factoidTxHash *Hash, credits int) *CBEntry {
	e := &CBEntry{}
	e.PublicKey = pubKey
	e.Type = TYPE_BUY	
	e.FactomTxHash = factoidTxHash
	e.Credits = credits
	
	return e
}

func (e *CBEntry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	
	data, _ := e.PublicKey.MarshalBinary()
	buf.Write(data)
	
	buf.Write([]byte{e.Type})
	if e.Type == TYPE_PAY {
		data, _ = e.EntryHash.MarshalBinary()
	} else if e.Type == TYPE_BUY {
		data, _ = e.FactomTxHash.MarshalBinary()
	}
	
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, e.Credits)	
	
	return buf.Bytes(), nil
}

func (e *CBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += e.PublicKey.MarshalledSize() 	// PublicKey	
	size += 1							// Type (byte)
	size += e.EntryHash.MarshalledSize()// Entry Hash or Factoid Trans Hash
	size += 4							// Credits (int)
	
	return size
}

func (e *CBEntry) UnmarshalBinary(data []byte) (err error) {

	e.PublicKey = new(Hash)
	e.PublicKey.UnmarshalBinary(data)
	data = data[e.PublicKey.MarshalledSize():]
	
	e.Type = data[0]
	
	if e.Type == TYPE_PAY {
		e.EntryHash = new(Hash)
		e.EntryHash.UnmarshalBinary(data)
		data = data[e.EntryHash.MarshalledSize():]
	} else if e.Type == TYPE_BUY {
		e.FactomTxHash = new(Hash)
		e.FactomTxHash.UnmarshalBinary(data)
		data = data[e.FactomTxHash.MarshalledSize():]
	}
	
	buf := bytes.NewBuffer(data[:4])
	binary.Read(buf, binary.BigEndian, &e.Credits)

	return nil
}
