package notaryapi

import (
	"bytes"
	"errors"
	
	"encoding/binary"
	"sync"
)

type FChain struct {
	ChainID 	*[]byte
	Blocks 		[]*FBlock
	BlockMutex 	sync.Mutex	
	NextBlockID uint64	
}

type FBlock struct {

	//Marshalized
	Header *FBlockHeader 
	FBEntries []*FBEntry
	Salt *Hash	

	//Not Marshalized
	Chain *FChain
	IsSealed bool
	FBHash *Hash 
	FBlockID uint64
}


type FBInfo struct {
	FBHash *Hash 
	FBlockID uint64
	BTCTxHash *Hash
}

	
type FBBatch struct {

	// FBlocks usually include 10 FBlocks, merkle root of which
	// is written into BTC. Only hash of each FBlock will be marshalled
	FBlocks []*FBlock	
	
	// BTCTxHash is the Tx hash returned from rpcclient.SendRawTransaction
	BTCTxHash *Hash	// use string or *btcwire.ShaHash ???
	
	// BTCTxOffset is the index of the TX in this BTC block
	BTCTxOffset int
	
	// BTCBlockHeight is the height of the block where this TX is stored in BTC
	BTCBlockHeight int32
	
	//BTCBlockHash is the hash of the block where this TX is stored in BTC
	BTCBlockHash *Hash	// use string or *btcwire.ShaHash ???
	
	// FBBatchMerkleRoot is the merkle root of a batch of 10 FactomBlocks
	// and is written into BTC as OP_RETURN data
	FBBatchMerkleRoot *Hash
}

func CreateFBlock(chain *FChain, prev *FBlock, capacity uint) (b *FBlock, err error) {
	if prev == nil && chain.NextBlockID != 0 {
		return nil, errors.New("Previous block cannot be nil")
	} else if prev != nil && chain.NextBlockID == 0 {
		return nil, errors.New("Origin block cannot have a parent block")
	}
	
	b = new(FBlock)
	
	var prevHash *Hash
	if prev == nil {
		prevHash = EmptyHash()
	} else {
		prevHash, err = CreateHash(prev)
	}
	
	b.Header = NewFBlockHeader(chain.NextBlockID, prevHash, FBlockVersion, uint32(0))
	
	b.Chain = chain

	
	b.FBEntries = make([]*FBEntry, 0, capacity)
	
	b.Salt = EmptyHash()
	
	b.IsSealed = false
	
	return b, err
}

// Add FBEntry from an Entry Block
func (fchain *FChain) AddFBEntry(eb *Block, hash *Hash) (err error) {
	fBlock := fchain.Blocks[len(fchain.Blocks)-1]
	
	fbEntry := NewFBEntry(hash, eb.Chain.ChainID)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(eb.Header.TimeStamp)) 	
	fbEntry.SetTimeStamp(b)
	
	
	fchain.BlockMutex.Lock()
	fBlock.FBEntries = append(fBlock.FBEntries, fbEntry) 
	fchain.BlockMutex.Unlock()

	return nil
	
	}


func (b *FBlock) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	if b.Header.MerkleRoot == nil {
		b.Header.MerkleRoot = b.calculateMerkleRoot()
	}

	b.Header.EntryCount = uint32(len(b.FBEntries))
	//fmt.Println("fblock.count=", b.Header.EntryCount)
	
	data, _ = b.Header.MarshalBinary()
	buf.Write(data)

	count := uint32(len(b.FBEntries))
	// need to get rid of count, duplicated with blockheader.entrycount
	binary.Write(&buf, binary.BigEndian, count)	
	for i := uint32(0); i < count; i = i + 1 {
		data, _ := b.FBEntries[i].MarshalBinary()
		buf.Write(data)
	}
	
	data, _ = b.Salt.MarshalBinary()
	buf.Write(data)
	
	return buf.Bytes(), err
}


func (b *FBlock) calculateMerkleRoot() *Hash {
	hashes := make([]*Hash, len(b.FBEntries))
	for i, entry := range b.FBEntries {
		data, _ := entry.MarshalBinary()
		hashes[i] = Sha(data)
	}
	
	merkle := BuildMerkleTreeStore(hashes)
	//merkle := BuildMerkleTreeStore(b.FBEntries)
	return merkle[len(merkle) - 1]
}


func (b *FBlock) MarshalledSize() uint64 {
	var size uint64 = 0
	
	size += b.Header.MarshalledSize()
	size += 4 // len(Entries) uint32
	size += b.Salt.MarshalledSize()
	
	for _, fbentry := range b.FBEntries {
		size += fbentry.MarshalledSize()
	}
	
	return 0
}

func (b *FBlock) UnmarshalBinary(data []byte) (err error) {
	fbh := new(FBlockHeader)
	fbh.UnmarshalBinary(data)
	b.Header = fbh
	data = data[fbh.MarshalledSize():]
	
	count, data := binary.BigEndian.Uint32(data[0:4]), data[4:]
	b.FBEntries = make([]*FBEntry, count)
	for i := uint32(0); i < count; i = i + 1 {
		b.FBEntries[i] = new(FBEntry)
		err = b.FBEntries[i].UnmarshalBinary(data)
		if err != nil { return }
		data = data[b.FBEntries[i].MarshalledSize():]
	}
	
	b.Salt = new(Hash)
	b.Salt.UnmarshalBinary(data)
	data = data[b.Salt.MarshalledSize():]
	
	return nil
}


func (b *FBBatch) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	
	count := len(b.FBlocks)
	binary.Write(&buf, binary.BigEndian, count)
	for _, fb := range b.FBlocks {
		data, _ := fb.FBHash.MarshalBinary()
		buf.Write(data)
	}

	data, _ = b.BTCTxHash.MarshalBinary()
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, b.BTCTxOffset)	
	binary.Write(&buf, binary.BigEndian, b.BTCBlockHeight)	

	data, _ = b.BTCBlockHash.MarshalBinary()
	buf.Write(data)

	data, _ = b.FBBatchMerkleRoot.MarshalBinary()
	buf.Write(data)
	
	return buf.Bytes(), err
}


func (b *FBBatch) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 4 + uint64(33 * len(b.FBlocks))	//FBlocks
	size += 33	//BTCTxHash
	size += 4	//BTCTxOffset
	size += 4 	//BTCBlockHeight
	size += 33	//BTCBlockHash
	size += 33	//FBBatchMerkleRoot
	
	return size	
}


func (b *FBBatch) UnmarshalBinary(data []byte) (err error) {
	count, data := binary.BigEndian.Uint32(data[0:4]), data[4:]
	b.FBlocks = make([]*FBlock, count)
	for i := uint32(0); i < count; i = i + 1 {
		b.FBlocks[i] = new(FBlock)
		err = b.FBlocks[i].FBHash.UnmarshalBinary(data)
		if err != nil { return }
		data = data[33:]
	}

	b.BTCTxHash = new(Hash)
	b.BTCTxHash.UnmarshalBinary(data[:33])	
	data = data[33:] 
	
	b.BTCTxOffset = int(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]
	
	b.BTCBlockHeight = int32(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]

	b.BTCBlockHash = new(Hash)
	b.BTCBlockHash.UnmarshalBinary(data[:33])	

	b.FBBatchMerkleRoot = new(Hash)
	b.FBBatchMerkleRoot.UnmarshalBinary(data[:33])	
	
	return nil
}


func (b *FBInfo) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	data, _ = b.FBHash.MarshalBinary()
	buf.Write(data)
	
	binary.Write(&buf, binary.BigEndian, b.FBlockID)	
		
	data, _ = b.BTCTxHash.MarshalBinary()
	buf.Write(data)
	
	return buf.Bytes(), err
}

func (b *FBInfo) MarshalledSize() uint64 {
	var size uint64 = 0
	size += 33	//FBHash
	size += 8	//FBlockID
	size += 33 	//BTCTxHash
	
	return size
}

func (b *FBInfo) UnmarshalBinary(data []byte) (err error) {

	b.FBHash = new(Hash)
	b.FBHash.UnmarshalBinary(data[:33])
	
	data = data[33:]
	b.FBlockID = binary.BigEndian.Uint64(data[:8])

	data = data[8:]
	b.BTCTxHash = new(Hash)
	b.BTCTxHash.UnmarshalBinary(data[:33])	
	
	
	return nil
}
