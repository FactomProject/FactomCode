    package chain

import (
	"fmt"
	"math/big"

	"github.com/FactomProject/FactomCode/factoid/state"
)


type StateTransition struct {
	coinbase, receiver []byte
	tx                 *Transaction
	value              *big.Int
	state              *state.State
	block              *Block

	cb, rec, sen *state.StateObject
}

func NewStateTransition(coinbase *state.StateObject, tx *Transaction, state *state.State, block *Block) *StateTransition {
	return &StateTransition{coinbase.Address(), tx.Recipient, tx, tx.Value, state, block, coinbase, nil, nil}
}

func (self *StateTransition) Coinbase() *state.StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetOrNewStateObject(self.coinbase)
	return self.cb
}
func (self *StateTransition) Sender() *state.StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetOrNewStateObject(self.tx.Sender())
	return self.sen
}
func (self *StateTransition) Receiver() *state.StateObject {
	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetOrNewStateObject(self.tx.Recipient)
	return self.rec
}

func (self *StateTransition) preCheck() (err error) {
	var (
		tx     = self.tx
		sender = self.Sender()
	)

	// Make sure this transaction's nonce is correct
	if sender.Nonce != tx.Nonce {
		return NonceError(tx.Nonce, sender.Nonce)
	}
	return nil
}

func (self *StateTransition) TransitionState() (err error) {
	chainlogger.Debugf("(~) %x\n", self.tx.Hash())

	defer func() {
		if r := recover(); r != nil {
			chainlogger.Infoln(r)
			err = fmt.Errorf("state transition err %v", r)
		}
	}()

	if err = self.preCheck(); err != nil {
		return
	}

	var (
		sender   = self.Sender()
		receiver *state.StateObject
	)

	sender.Nonce += 1
	receiver = self.Receiver()
	
	return self.transferValue(sender, receiver)
}

func (self *StateTransition) transferValue(sender, receiver *state.StateObject) error {
	totalValue := new(big.Int).Add(self.value, self.tx.Fee)
	if sender.Balance.Cmp(totalValue) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, totalValue)
	}

	sender.SubAmount(totalValue)
	receiver.AddAmount(self.value)

	return nil
}


