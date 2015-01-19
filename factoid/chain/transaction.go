package chain

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/FactomProject/FactomCode/factoid/crypto"
	"github.com/FactomProject/FactomCode/factoid/util"
	//	"github.com/FactomProject/FactomCode/factoid/state"

	"github.com/obscuren/secp256k1-go"
)

type Transaction struct {
	Nonce     uint64
	Recipient []byte
	Value     *big.Int
	Fee       *big.Int
	v         byte
	r, s      []byte
}

//func NewTransaction(to []byte, value *big.Int) *Transaction {
//	return &Transaction{Recipient: to, Value: value, Fee: util.Big0}
//}

func NewTransaction(to []byte, value *big.Int, fee *big.Int) *Transaction {
	return &Transaction{Recipient: to, Value: value, Fee: fee}
}

func NewTransactionFromBytes(data []byte) *Transaction {
	tx := &Transaction{}
	tx.RlpDecode(data)

	return tx
}

func NewTransactionFromValue(val *util.Value) *Transaction {
	tx := &Transaction{}
	tx.RlpValueDecode(val)

	return tx
}

func (tx *Transaction) Hash() []byte {
	data := []interface{}{tx.Nonce, tx.Recipient, tx.Value}

	return crypto.Sha3Bin(util.NewValue(data).Encode())
}

func (tx *Transaction) Signature(key []byte) []byte {
	hash := tx.Hash()

	sig, _ := secp256k1.Sign(hash, key)

	return sig
}

func (tx *Transaction) PublicKey() []byte {
	hash := tx.Hash()

	// TODO
	r := util.LeftPadBytes(tx.r, 32)
	s := util.LeftPadBytes(tx.s, 32)

	sig := append(r, s...)
	sig = append(sig, tx.v-27)

	pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

	return pubkey
}

func (tx *Transaction) Sender() []byte {
	pubkey := tx.PublicKey()

	// Validate the returned key.
	// Return nil if public key isn't in full format
	if pubkey[0] != 4 {
		return nil
	}

	return crypto.Sha3Bin(pubkey[1:])[12:]
}

func (tx *Transaction) Sign(privk []byte) error {

	sig := tx.Signature(privk)

	tx.r = sig[:32]
	tx.s = sig[32:64]
	tx.v = sig[64] + 27

	return nil
}

func (tx *Transaction) RlpData() interface{} {
	data := []interface{}{tx.Nonce, tx.Recipient, tx.Value}

	// TODO Remove prefixing zero's

	return append(data, tx.v, new(big.Int).SetBytes(tx.r).Bytes(), new(big.Int).SetBytes(tx.s).Bytes())
}

func (tx *Transaction) RlpValue() *util.Value {
	return util.NewValue(tx.RlpData())
}

func (tx *Transaction) RlpEncode() []byte {
	return tx.RlpValue().Encode()
}

func (tx *Transaction) RlpDecode(data []byte) {
	tx.RlpValueDecode(util.NewValueFromBytes(data))
}

func (tx *Transaction) RlpValueDecode(decoder *util.Value) {
	tx.Nonce = decoder.Get(0).Uint()
	tx.Fee = decoder.Get(1).BigInt()
	tx.Recipient = decoder.Get(2).Bytes()
	tx.Value = decoder.Get(3).BigInt()
	tx.v = byte(decoder.Get(4).Uint())

	tx.r = decoder.Get(5).Bytes()
	tx.s = decoder.Get(6).Bytes()
}

func (tx *Transaction) String() string {
	return fmt.Sprintf(`
	TX(%x)
	From:     %x
	To:       %x
	Nonce:    %v
	Value:    %v
	Fee:	  %v
	V:        0x%x
	R:        0x%x
	S:        0x%x
	`,
		tx.Hash(),
		tx.Sender(),
		tx.Recipient,
		tx.Nonce,
		tx.Fee,
		tx.Value,
		tx.v,
		tx.r,
		tx.s)
}

type Receipt struct {
	Tx            *Transaction
	PostState     []byte
	CumulativeFee *big.Int
}
type Receipts []*Receipt

func NewRecieptFromValue(val *util.Value) *Receipt {
	r := &Receipt{}
	r.RlpValueDecode(val)

	return r
}

func (self *Receipt) RlpValueDecode(decoder *util.Value) {
	self.Tx = NewTransactionFromValue(decoder.Get(0))
	self.PostState = decoder.Get(1).Bytes()
	self.CumulativeFee = decoder.Get(2).BigInt()
}

func (self *Receipt) RlpData() interface{} {
	return []interface{}{self.Tx.RlpData(), self.PostState, self.CumulativeFee}
}

func (self *Receipt) String() string {
	return fmt.Sprintf(`
	R
	Tx:[                 %v]
	PostState:           0x%x
	CumulativeFee:   %v
	`,
		self.Tx,
		self.PostState,
		self.CumulativeFee)
}

func (self *Receipt) Cmp(other *Receipt) bool {
	if bytes.Compare(self.PostState, other.PostState) != 0 {
		return false
	}

	return true
}

// Transaction slice type for basic sorting
type Transactions []*Transaction

func (s Transactions) Len() int      { return len(s) }
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type TxByNonce struct{ Transactions }

func (s TxByNonce) Less(i, j int) bool {
	return s.Transactions[i].Nonce < s.Transactions[j].Nonce
}
