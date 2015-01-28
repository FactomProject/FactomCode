package notaryapi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

type CChain struct {
	ChainID      *Hash
	Name         [][]byte
	Blocks       []*CBlock
	CurrentBlock *CBlock
	BlockMutex   sync.Mutex
	NextBlockID  uint64
}

type CBlock struct {
	//Marshalized
	Header    *CBlockHeader
	CBEntries []CBEntry //Interface
	//Not Marshalized
	CBHash   *Hash
	Salt     *Hash
	Chain    *CChain
	IsSealed bool
}

type CBInfo struct {
	CBHash     *Hash
	FBHash     *Hash
	FBBlockNum uint64
	ChainID    *Hash
}

func CreateCBlock(chain *CChain, prev *CBlock, cap uint) (b *CBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}

	b = new(CBlock)

	var prevHash *Hash
	if prev == nil {
		prevHash = NewHash()
	} else {
		prevHash, err = CreateHash(prev)
	}

	b.Header = NewCBlockHeader(chain.NextBlockID, prevHash, NewHash())
	b.Chain = chain
	b.CBEntries = make([]CBEntry, 0, cap)
	b.Salt = NewHash()
	b.IsSealed = false

	return b, err
}

func (b *CBlock) AddCBEntry(e CBEntry) (err error) {
	b.CBEntries = append(b.CBEntries, e)
	return
}

func (b *CBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.Header.MarshalBinary()
	buf.Write(data)

	count := uint64(len(b.CBEntries))
	binary.Write(&buf, binary.BigEndian, count)
	for i := uint64(0); i < count; i++ {
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

	b.CBEntries = make([]CBEntry, count)
	for i := uint64(0); i < count; i++ {
		if data[0] == TYPE_BUY {
			b.CBEntries[i] = new(BuyCBEntry)
		} else if data[0] == TYPE_PAY_CHAIN {
			b.CBEntries[i] = new(PayChainCBEntry)
		} else if data[0] == TYPE_PAY_ENTRY {
			b.CBEntries[i] = new(PayEntryCBEntry)
		}
		err = b.CBEntries[i].UnmarshalBinary(data)
		if err != nil {
			return
		}
		data = data[b.CBEntries[i].MarshalledSize():]
	}

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
	size += 33 //b.EBHash
	size += 33 //b.FBHash
	size += 8  //b.FBBlockNum
	size += 33 //b.ChainID

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
	size += 33 //b.ChainID
	size += 8  //Name length
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

	for i := uint64(0); i < count; i++ {
		length := binary.BigEndian.Uint64(data[0:8])
		data = data[8:]
		b.Name[i] = data[:length]
		data = data[length:]
	}

	return nil
}

//-----------------------
type CBlockHeader struct {
	BlockID       uint64
	PrevBlockHash *Hash
	TimeStamp     int64
	EntryCount    uint32
}

func (b *CBlockHeader) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, b.BlockID)
	data, err = b.PrevBlockHash.MarshalBinary()
	if err != nil {
		return
	}

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
		PrevBlockHash: prevHash,
		TimeStamp:     time.Now().Unix(),
		BlockID:       blockId,
	}
}

//---------------------------------------------------------------
// Three types of entries (transactions) for Entry Credit Block
//---------------------------------------------------------------
const (
	TYPE_BUY uint8 = iota
	TYPE_PAY_ENTRY
	TYPE_PAY_CHAIN
)

type CBEntry interface {
	Type() byte
	PublicKey() *Hash
	Credits() int32
	MarshalBinary() ([]byte, error)
	MarshalledSize() uint64
	UnmarshalBinary(data []byte) (err error)
}

type BuyCBEntry struct {
	entryType    byte
	publicKey    *Hash
	credits      int32
	CBEntry      //interface
	FactomTxHash *Hash
}

type PayEntryCBEntry struct {
	entryType byte
	publicKey *Hash
	credits   int32
	CBEntry   //interface
	EntryHash *Hash
	TimeStamp int64
}

type PayChainCBEntry struct {
	entryType        byte
	publicKey        *Hash
	credits          int32
	CBEntry          //interface
	EntryHash        *Hash
	ChainIDHash      *Hash
	EntryChainIDHash *Hash //Hash(EntryHash+ChainIDHash)
}

type ECBalance struct {
	PublicKey *Hash
	Credits   int32
}

func NewPayEntryCBEntry(pubKey *Hash, entryHash *Hash, credits int32,
	timeStamp int64) *PayEntryCBEntry {
	e := &PayEntryCBEntry{}
	e.publicKey = pubKey
	e.entryType = TYPE_PAY_ENTRY
	e.credits = credits
	e.EntryHash = entryHash
	e.TimeStamp = timeStamp

	return e
}

func NewPayChainCBEntry(pubKey *Hash, entryHash *Hash, credits int32,
	chainIDHash *Hash, entryChainIDHash *Hash) *PayChainCBEntry {
	e := &PayChainCBEntry{}
	e.publicKey = pubKey
	e.entryType = TYPE_PAY_CHAIN
	e.credits = credits
	e.EntryHash = entryHash
	e.ChainIDHash = chainIDHash
	e.EntryChainIDHash = entryChainIDHash

	return e
}

func NewBuyCBEntry(pubKey *Hash, factoidTxHash *Hash,
	credits int32) *BuyCBEntry {
	e := &BuyCBEntry{}
	e.publicKey = pubKey
	e.entryType = TYPE_BUY
	e.FactomTxHash = factoidTxHash
	e.credits = credits

	return e
}

func (e *BuyCBEntry) Type() byte {
	return e.entryType
}

func (e *BuyCBEntry) PublicKey() *Hash {
	return e.publicKey
}

func (e *BuyCBEntry) Credits() int32 {
	return e.credits
}

func (e *BuyCBEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{e.entryType})

	data, err = e.publicKey.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)
	binary.Write(&buf, binary.BigEndian, e.Credits())

	data, err = e.FactomTxHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

func (e *BuyCBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 1                               // Type (byte)
	size += e.publicKey.MarshalledSize()    // PublicKey
	size += 4                               // Credits (int32)
	size += e.FactomTxHash.MarshalledSize() // Factoid Trans Hash

	return size
}

func (e *BuyCBEntry) UnmarshalBinary(data []byte) (err error) {
	e.entryType, data = data[0], data[1:]
	e.publicKey = new(Hash)

	e.publicKey.UnmarshalBinary(data)
	data = data[e.publicKey.MarshalledSize():]

	buf, data := bytes.NewBuffer(data[:4]), data[4:]
	binary.Read(buf, binary.BigEndian, &e.credits)

	e.FactomTxHash = new(Hash)
	e.FactomTxHash.UnmarshalBinary(data)

	return nil
}

func (e *PayEntryCBEntry) Type() byte {
	return e.entryType
}

func (e *PayEntryCBEntry) PublicKey() *Hash {
	return e.publicKey
}

func (e *PayEntryCBEntry) Credits() int32 {
	return e.credits
}

func (e *PayEntryCBEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{e.entryType})

	data, err = e.publicKey.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, e.credits)

	data, err = e.EntryHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, e.TimeStamp)

	return buf.Bytes(), nil
}

func (e *PayEntryCBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 1                            // Type (byte)
	size += e.publicKey.MarshalledSize() // PublicKey
	size += 4                            // Credits (int32)
	size += e.EntryHash.MarshalledSize() // Entry Hash
	size += 8                            //	TimeStamp int64

	return size
}

func (e *PayEntryCBEntry) UnmarshalBinary(data []byte) (err error) {
	e.entryType, data = data[0], data[1:]

	e.publicKey = new(Hash)
	e.publicKey.UnmarshalBinary(data)
	data = data[e.publicKey.MarshalledSize():]

	buf, data := bytes.NewBuffer(data[:4]), data[4:]
	binary.Read(buf, binary.BigEndian, &e.credits)

	e.EntryHash = new(Hash)
	e.EntryHash.UnmarshalBinary(data)
	data = data[e.EntryHash.MarshalledSize():]

	buf = bytes.NewBuffer(data[:4])
	binary.Read(buf, binary.BigEndian, &e.TimeStamp)

	return nil
}

func (e *PayChainCBEntry) Type() byte {
	return e.entryType
}

func (e *PayChainCBEntry) PublicKey() *Hash {
	return e.publicKey
}

func (e *PayChainCBEntry) Credits() int32 {
	return e.credits
}

func (e *PayChainCBEntry) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	buf.Write([]byte{e.entryType})

	data, err = e.publicKey.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, e.credits)

	data, err = e.EntryHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = e.ChainIDHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	data, err = e.EntryChainIDHash.MarshalBinary()
	if err != nil {
		return
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

func (e *PayChainCBEntry) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 1                                   // Type (byte)
	size += e.publicKey.MarshalledSize()        // PublicKey
	size += 4                                   // Credits (int32)
	size += e.EntryHash.MarshalledSize()        // Entry Hash
	size += e.ChainIDHash.MarshalledSize()      // ChainID Hash
	size += e.EntryChainIDHash.MarshalledSize() // EntryChainID Hash

	return size
}

func (e *PayChainCBEntry) UnmarshalBinary(data []byte) (err error) {
	e.entryType, data = data[0], data[1:]

	e.publicKey = new(Hash)
	e.publicKey.UnmarshalBinary(data)
	data = data[e.publicKey.MarshalledSize():]

	buf, data := bytes.NewBuffer(data[:4]), data[4:]
	binary.Read(buf, binary.BigEndian, &e.credits)

	e.EntryHash = new(Hash)
	e.EntryHash.UnmarshalBinary(data)
	data = data[e.EntryHash.MarshalledSize():]

	e.ChainIDHash = new(Hash)
	e.ChainIDHash.UnmarshalBinary(data)
	data = data[e.ChainIDHash.MarshalledSize():]

	e.EntryChainIDHash = new(Hash)
	e.EntryChainIDHash.UnmarshalBinary(data)

	return nil
}
