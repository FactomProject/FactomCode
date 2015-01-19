package state

import (
	"fmt"
	"math/big"

	//	"github.com/FactomProject/FactomCode/factoid/crypto"
	"github.com/FactomProject/FactomCode/factoid/db"
	"github.com/FactomProject/FactomCode/factoid/trie"
	"github.com/FactomProject/FactomCode/factoid/util"
)

type Storage map[string]*util.Value

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		// XXX Do we need a 'value' copy or is this sufficient?
		cpy[key] = value
	}

	return cpy
}

type StateObject struct {
	// Address of the object
	address []byte
	// Shared attributes
	Balance *big.Int
	Nonce   uint64
	// Contract related attributes
	State *State

	storage Storage

	// Mark for deletion
	// When an object is marked for deletion it will be delete from the trie
	// during the "update" phase of the state transition
	remove bool
}

func (self *StateObject) Reset() {
	self.storage = make(Storage)
	self.State.Reset()
}

func NewStateObject(addr []byte) *StateObject {
	// This to ensure that it has 20 bytes (and not 0 bytes), thus left or right pad doesn't matter.
	address := util.Address(addr)

	object := &StateObject{address: address, Balance: new(big.Int)}
	object.State = New(trie.New(db.Db, ""))
	object.storage = make(Storage)

	return object
}

func NewStateObjectFromBytes(address, data []byte) *StateObject {
	object := &StateObject{address: address}
	object.RlpDecode(data)

	return object
}

func (self *StateObject) MarkForDeletion() {
	self.remove = true
	statelogger.DebugDetailf("%x: #%d %v (deletion)\n", self.Address(), self.Nonce, self.Balance)
}

func (c *StateObject) GetAddr(addr []byte) *util.Value {
	return util.NewValueFromBytes([]byte(c.State.Trie.Get(string(addr))))
}

func (c *StateObject) SetAddr(addr []byte, value interface{}) {
	c.State.Trie.Update(string(addr), string(util.NewValue(value).Encode()))
}

func (self *StateObject) GetStorage(key *big.Int) *util.Value {
	return self.getStorage(key.Bytes())
}
func (self *StateObject) SetStorage(key *big.Int, value *util.Value) {
	self.setStorage(key.Bytes(), value)
}

func (self *StateObject) getStorage(k []byte) *util.Value {
	key := util.LeftPadBytes(k, 32)

	value := self.storage[string(key)]
	if value == nil {
		value = self.GetAddr(key)

		if !value.IsNil() {
			self.storage[string(key)] = value
		}
	}

	return value
}

func (self *StateObject) setStorage(k []byte, value *util.Value) {
	key := util.LeftPadBytes(k, 32)
	self.storage[string(key)] = value.Copy()
}

// Iterate over each storage address and yield callback
func (self *StateObject) EachStorage(cb trie.EachCallback) {
	// First loop over the uncommit/cached values in storage
	for key, value := range self.storage {
		// XXX Most iterators Fns as it stands require encoded values
		encoded := util.NewValue(value.Encode())
		cb(key, encoded)
	}

	it := self.State.Trie.NewIterator()
	it.Each(func(key string, value *util.Value) {
		// If it's cached don't call the callback.
		if self.storage[key] == nil {
			cb(key, value)
		}
	})
}

func (self *StateObject) Sync() {
	for key, value := range self.storage {
		if value.Len() == 0 {
			//fmt.Printf("deleting %x %x 0x%x\n", self.Address(), []byte(key), data)
			self.State.Trie.Delete(string(key))
			continue
		}

		self.SetAddr([]byte(key), value)
	}

	valid, t2 := trie.ParanoiaCheck(self.State.Trie)
	if !valid {
		statelogger.Infof("Warn: PARANOIA: Different state storage root during copy %x vs %x\n", self.State.Trie.Root, t2.Root)

		self.State.Trie = t2
	}
}

func (c *StateObject) AddAmount(amount *big.Int) {
	c.SetBalance(new(big.Int).Add(c.Balance, amount))

	statelogger.Debugf("%x: #%d %v (+ %v)\n", c.Address(), c.Nonce, c.Balance, amount)
}

func (c *StateObject) SubAmount(amount *big.Int) {
	c.SetBalance(new(big.Int).Sub(c.Balance, amount))

	statelogger.Debugf("%x: #%d %v (- %v)\n", c.Address(), c.Nonce, c.Balance, amount)
}

func (c *StateObject) SetBalance(amount *big.Int) {
	c.Balance = amount
}

func (self *StateObject) Copy() *StateObject {
	stateObject := NewStateObject(self.Address())
	stateObject.Balance.Set(self.Balance)
	stateObject.Nonce = self.Nonce
	if self.State != nil {
		stateObject.State = self.State.Copy()
	}
	stateObject.storage = self.storage.Copy()
	stateObject.remove = self.remove

	return stateObject
}

func (self *StateObject) Set(stateObject *StateObject) {
	*self = *stateObject
}

//
// Attribute accessors
//

func (c *StateObject) N() *big.Int {
	return big.NewInt(int64(c.Nonce))
}

// Returns the address of the contract/account
func (c *StateObject) Address() []byte {
	return c.address
}

// To satisfy ClosureRef
func (self *StateObject) Object() *StateObject {
	return self
}

// Debug stuff
func (self *StateObject) CreateOutputForDiff() {
	fmt.Printf("%x %x %x %x\n", self.Address(), self.State.Root(), self.Balance.Bytes(), self.Nonce)
	self.EachStorage(func(addr string, value *util.Value) {
		fmt.Printf("%x %x\n", addr, value.Bytes())
	})
}

//
// Encoding
//

// State object encoding methods
func (c *StateObject) RlpEncode() []byte {
	var root interface{}
	if c.State != nil {
		root = c.State.Trie.Root
	} else {
		root = ""
	}

	return util.Encode([]interface{}{c.Nonce, c.Balance, root})
}

func (c *StateObject) RlpDecode(data []byte) {
	decoder := util.NewValueFromBytes(data)

	c.Nonce = decoder.Get(0).Uint()
	c.Balance = decoder.Get(1).BigInt()
	c.State = New(trie.New(db.Db, decoder.Get(2).Interface()))
	c.storage = make(map[string]*util.Value)
}
