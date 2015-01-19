package chain

import (
	"bytes"
	"math/big"

	"github.com/FactomProject/FactomCode/factoid/db"
	"github.com/FactomProject/FactomCode/factoid/log"
	"github.com/FactomProject/FactomCode/factoid/util"
)

var FChain *BlockChain = NewBlockChain()

var chainlogger = log.NewLogger("CHAIN")

type BlockChain struct {
	genesisBlock *Block

	LastBlockNumber uint64

	CurrentBlock  *Block
	LastBlockHash []byte
}

func NewBlockChain() *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = NewBlockFromBytes(util.Encode(Genesis))

	bc.setLastBlock()

	return bc
}

func (bc *BlockChain) Genesis() *Block {
	return bc.genesisBlock
}

func (bc *BlockChain) NewBlock(coinbase []byte) *Block {
	var root interface{}
	hash := ZeroHash256

	if bc.CurrentBlock != nil {
		root = bc.CurrentBlock.state.Trie.Root
		hash = bc.LastBlockHash
	}

	block := CreateBlock(
		root,
		hash,
		coinbase,
		nil)

	parent := bc.CurrentBlock
	if parent != nil {
		block.Number = new(big.Int).Add(bc.CurrentBlock.Number, util.Big1)
	}

	return block
}

func (bc *BlockChain) HasBlock(hash []byte) bool {
	data, _ := db.Db.Get(hash)
	return len(data) != 0
}

// TODO: At one point we might want to save a block by prevHash in the db to optimise this...
func (bc *BlockChain) HasBlockWithPrevHash(hash []byte) bool {
	block := bc.CurrentBlock

	for ; block != nil; block = bc.GetBlock(block.PrevHash) {
		if bytes.Compare(hash, block.PrevHash) == 0 {
			return true
		}
	}
	return false
}

func (bc *BlockChain) GenesisBlock() *Block {
	return bc.genesisBlock
}

func (self *BlockChain) GetChainHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		chain = append(chain, block.Hash())

		if block.Number.Cmp(util.Big0) <= 0 {
			break
		}

		block = self.GetBlock(block.PrevHash)
	}

	return
}

func AddTestNetFunds(block *Block) {
	for _, addr := range []string{
		"36bf454e58a27729f818fd34a7e26bc1a2b2767f",
		"05413a2e2188e0b921b18766a3d1e4c591b30dce",
		"90a2f80f9f48164eebe7bcf23b56019216ace826",
		"18243092ae005f0b4d1253d42ade1bb15be75ab3",
	} {
		codedAddr := util.Hex2Bytes(addr)
		account := block.state.GetAccount(codedAddr)
		account.Balance = util.Big("10000")
		block.state.UpdateStateObject(account)
	}
}

func (bc *BlockChain) setLastBlock() {
	// Prep genesis
	AddTestNetFunds(bc.genesisBlock)

	data, _ := db.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := NewBlockFromBytes(data)
		bc.CurrentBlock = block
		bc.LastBlockHash = block.Hash()
		bc.LastBlockNumber = block.Number.Uint64()

	} else {
		bc.genesisBlock.state.Trie.Sync()
		// Prepare the genesis block
		bc.Add(bc.genesisBlock)
		bc.CurrentBlock = bc.genesisBlock
	}

	chainlogger.Infof("Last block (#%d) %x\n", bc.LastBlockNumber, bc.CurrentBlock.Hash())
}

// Add a block to the chain and record addition information
func (bc *BlockChain) Add(block *Block) {
	bc.CurrentBlock = block
	bc.LastBlockHash = block.Hash()

	encodedBlock := block.RlpEncode()
	db.Db.Put(block.Hash(), encodedBlock)
	db.Db.Put([]byte("LastBlock"), encodedBlock)
}

func (bc *BlockChain) GetBlock(hash []byte) *Block {
	data, _ := db.Db.Get(hash)
	if len(data) == 0 {
		return nil
	}

	return NewBlockFromBytes(data)
}

func (self *BlockChain) GetBlockByNumber(num uint64) *Block {
	block := self.CurrentBlock
	for ; block != nil; block = self.GetBlock(block.PrevHash) {
		if block.Number.Uint64() == num {
			break
		}
	}

	if block != nil && block.Number.Uint64() == 0 && num != 0 {
		return nil
	}

	return block
}
