package chain

import (
	"bytes"
	"fmt"
	"math/big"
	_ "strconv"
	"time"

	"github.com/FactomProject/FactomCode/factoid/crypto"
	"github.com/FactomProject/FactomCode/factoid/db"
	"github.com/FactomProject/FactomCode/factoid/state"
	"github.com/FactomProject/FactomCode/factoid/trie"
	"github.com/FactomProject/FactomCode/factoid/util"
)

type Blocks []*Block 

const MAX_TX_NUM = 10000

type Block struct {
	PrevHash util.Bytes
	Coinbase []byte
	// Block Trie state
	state *state.State
	// Creation time
	Time int64
	// The block number
	Number *big.Int

	TotalFee *big.Int
	// Block Nonce for verification
	Nonce util.Bytes
	// List of transactions and/or contracts
	transactions []*Transaction
	receipts     []*Receipt
	TxSha        []byte
}

func NewBlockFromBytes(raw []byte) *Block {
	block := &Block{}
	block.RlpDecode(raw)

	return block
}

// New block takes a raw encoded string 
func NewBlockFromRlpValue(rlpValue *util.Value) *Block {
	block := &Block{}
	block.RlpValueDecode(rlpValue)

	return block
}

func CreateBlock(root interface{},
	prevHash []byte,
	base []byte,
	Nonce []byte) *Block {

	block := &Block{
		PrevHash: prevHash,
		Coinbase: base,
		Nonce: Nonce,
		Time:  time.Now().Unix(),
		TotalFee: new(big.Int),
	}

	block.state = state.New(trie.New(db.Db, root))
	
	block.transactions = make([]*Transaction, 0, 100)

	return block
}

// Returns a hash of the block
func (block *Block) Hash() util.Bytes {
	return crypto.Sha3Bin(util.NewValue(block.header()).Encode())
}

func (block *Block) HashNoNonce() []byte {
	return crypto.Sha3Bin(util.Encode([]interface{}{block.PrevHash,
		block.Coinbase, block.state.Trie.Root,
		block.TxSha, block.Number,
		block.Time}))
}

func (block *Block) State() *state.State {
	return block.state
}

func (block *Block) SetState(st *state.State) {
	block.state = st
}

func (block *Block) Transactions() []*Transaction {
	return block.transactions
}

func (self *Block) GetTransaction(hash []byte) *Transaction {
	for _, receipt := range self.receipts {
		if bytes.Compare(receipt.Tx.Hash(), hash) == 0 {
			return receipt.Tx
		}
	}

	return nil
}

// Sync the block's state and contract respectively
func (block *Block) Sync() {
	block.state.Sync()
}

func (block *Block) Undo() {
	// Sync the block state itself
	block.state.Reset()
}

/////// Block Encoding
func (block *Block) rlpReceipts() interface{} {
	// Marshal the transactions of this block
	encR := make([]interface{}, len(block.receipts))
	for i, r := range block.receipts {
		// Cast it to a string (safe)
		encR[i] = r.RlpData()
	}

	return encR
}


func (self *Block) SetReceipts(receipts []*Receipt, txs []*Transaction) {
	self.receipts = receipts
	self.setTransactions(txs)
}

func (block *Block) setTransactions(txs []*Transaction) {
	block.transactions = txs
}

func CreateTxSha(receipts Receipts) (sha []byte) {
	trie := trie.New(db.Db, "")
	for i, receipt := range receipts {
		trie.Update(string(util.NewValue(i).Encode()), string(util.NewValue(receipt.RlpData()).Encode()))
	}

	switch trie.Root.(type) {
	case string:
		sha = []byte(trie.Root.(string))
	case []byte:
		sha = trie.Root.([]byte)
	default:
		panic(fmt.Sprintf("invalid root type %T", trie.Root))
	}

	return sha
}

func (self *Block) SetTxHash(receipts Receipts) {
	self.receipts = receipts	//added
	self.TxSha = CreateTxSha(receipts)
}

func (block *Block) Value() *util.Value {
	return util.NewValue([]interface{}{block.header(), block.rlpReceipts()})
}

func (block *Block) RlpEncode() []byte {
	// Encode a slice interface which contains the header and the list of
	// transactions.
	return block.Value().Encode()
}

func (block *Block) RlpDecode(data []byte) {
	rlpValue := util.NewValueFromBytes(data)
	block.RlpValueDecode(rlpValue)
}

func (block *Block) RlpValueDecode(decoder *util.Value) {
	header := decoder.Get(0)

	block.PrevHash = header.Get(0).Bytes()
	block.Coinbase = header.Get(1).Bytes()
	block.state = state.New(trie.New(db.Db, header.Get(2).Val))
	block.TxSha = header.Get(3).Bytes()
	block.Number = header.Get(4).BigInt()
	//fmt.Printf("#%v : %x\n", block.Number, block.Coinbase)
	block.TotalFee = header.Get(5).BigInt()
	block.Time = int64(header.Get(6).BigInt().Uint64())
	block.Nonce = header.Get(7).Bytes()

}


func (block *Block) GetRoot() interface{} {
	return block.state.Trie.Root
}

func (self *Block) Receipts() []*Receipt {
	return self.receipts
}

func (block *Block) header() []interface{} {
	return []interface{}{
		// Sha of the previous block
		block.PrevHash,
		// Coinbase address
		block.Coinbase,
		// root state
		block.state.Trie.Root,
		// Sha of tx
		block.TxSha,
		// The block number
		block.Number,
		// Block total tx fee
		block.TotalFee,
		// Time the block was found?
		block.Time,
		// Block's Nonce for validation
		block.Nonce,
	}
}

func (block *Block) String() string {
	return fmt.Sprintf(`
	BLOCK(%x): 	
	PrevHash:   %x
	Coinbase:   %x
	Root:       %x
	TxSha:      %x
	Number:     %v
	TotalFee:	%v
	Time:       %v
	Nonce:      %x
	NumTx:      %v
`,
		block.Hash(),
		block.PrevHash,
		block.Coinbase,
		block.state.Trie.Root,
		block.TxSha,
		block.Number,
		block.TotalFee,
		block.Time,
		block.Nonce,
		len(block.transactions),
	)
}


func (block *Block) AddTransaction(tx *Transaction) {
	err := block.ValidateTransaction(tx)
	if err == nil {
		block.transactions = append(block.transactions, tx)
	} else {
		fmt.Println(err.Error())
	}
}


func (block *Block) ValidateTransaction(tx *Transaction) error {
	// Get the last block so we can retrieve the sender and receiver from
	// the merkle trie
	if block == nil {
		return fmt.Errorf("[TXPL] No last block on the block chain")
	}

	if len(tx.Recipient) != 0 && len(tx.Recipient) != 20 {
		return fmt.Errorf("[TXPL] Invalid recipient. len = %d", len(tx.Recipient))
	}

	// Get the sender
	sender := FChain.CurrentBlock.State().GetAccount(tx.Sender())
	
	fmt.Println("sender=", util.Bytes2Hex(tx.Sender()), ", rec=%", util.Bytes2Hex(tx.Recipient),	", balance=", sender.Balance, ", payment=", tx.Value, ", fee=", tx.Fee)

	totalValue := new(big.Int).Add(tx.Value, tx.Fee)
	if sender.Balance.Cmp(totalValue) < 0 {
		return fmt.Errorf("[TXPL] Insufficient amount in sender's (%x) account, balance=%v, payment=%v, tx.fee=%v", tx.Sender(), sender.Balance, tx.Value, tx.Fee)
	}

	return nil
}

