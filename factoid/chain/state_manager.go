package chain

import (
	//	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/FactomProject/FactomCode/factoid/log"
	"github.com/FactomProject/FactomCode/factoid/state"
	"github.com/FactomProject/FactomCode/factoid/util"
)

var BlockReward *big.Int = big.NewInt(1)

var statelogger = log.NewLogger("STATE")

type StateManager struct {
	mutex sync.Mutex
	bc    *BlockChain
	// The managed states
	// Transiently state. The trans state isn't ever saved, validated and
	// it could be used for setting account nonces without effecting
	// the main states.
	transState *state.State

	// The last attempted block is mainly used for debugging purposes
	// This does not have to be a valid block and will be set during
	// 'Process' & canonical validation.
	lastAttemptedBlock *Block
}

func NewStateManager() *StateManager {
	sm := &StateManager{
		bc: FChain,
	}
	sm.transState = FChain.CurrentBlock.State().Copy()

	return sm
}

func (sm *StateManager) CurrentState() *state.State {
	return sm.bc.CurrentBlock.State()
}

func (sm *StateManager) TransState() *state.State {
	return sm.transState
}

func (self *StateManager) ProcessTransactions(coinbase *state.StateObject, state *state.State, block, parent *Block, txs Transactions) (Receipts, Transactions, Transactions, error) {
	var (
		receipts           Receipts
		handled, unhandled Transactions
		totalFee           = big.NewInt(0)
		err                error
	)

	//	fmt.Println("block=", block)
	//	fmt.Println("parent=", parent)
	//	fmt.Println("len(txs)=", len(txs))

	for i, tx := range txs {

		cb := state.GetStateObject(coinbase.Address())
		st := NewStateTransition(cb, tx, state, block)
		err = st.TransitionState()
		if err != nil {
			fmt.Println("Process TX Error: ", err.Error(), ", tx=", tx)
			statelogger.Infoln(err)
			switch {
			case IsNonceErr(err):
				err = nil // ignore error
				continue
			default:
				statelogger.Infoln(err)
				err = nil
			}
		}

		// Update the state with pending changes
		state.Update()

		totalFee.Add(totalFee, tx.Fee)
		receipt := &Receipt{tx, util.CopyBytes(state.Root().([]byte)), totalFee}

		if i < len(block.Receipts()) {
			original := block.Receipts()[i]
			if !original.Cmp(receipt) {

				err := fmt.Errorf("#%d receipt failed (r) %v ~ %x  <=>  (c) %v ~ %x (%x...)", i+1, original.CumulativeFee, original.PostState[0:4], receipt.CumulativeFee, receipt.PostState[0:4], receipt.Tx.Hash()[0:4])

				return nil, nil, nil, err
			}
		}

		receipts = append(receipts, receipt)
		handled = append(handled, tx)
	}

	block.TotalFee = totalFee //??? parent.TotalFee

	fmt.Println("receipts.len=", len(receipts), ", receipts=", receipts, ", totalFee=", totalFee)

	return receipts, handled, unhandled, err
}

func (sm *StateManager) Process(block *Block) (err error) {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.bc.HasBlock(block.Hash()) {
		return nil
	}

	if !sm.bc.HasBlock(block.PrevHash) {
		return ParentError(block.PrevHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.PrevHash)
		state  = parent.State()
	)

	defer state.Reset()

	receipts, err := sm.ApplyDiff(state, parent, block)
	if err != nil {
		return err
	}

	block.SetTxHash(receipts)

	// Block validation
	if err = sm.ValidateBlock(block); err != nil {
		statelogger.Errorln("Error validating block:", err)
		return err
	}

	if err = sm.AccumelateRewards(state, block, parent); err != nil {
		statelogger.Errorln("Error accumulating reward", err)
		return err
	}

	state.Update()
	block.SetState(state.Copy()) //added

	if !block.State().Cmp(state) {
		err = fmt.Errorf("Invalid merkle root.\nrec: %x\nis:  %x", block.State().Trie.Root, state.Trie.Root)
		return
	}

	// Sync the current block's state to the database and cancelling out the deferred Undo
	state.Sync()

	// Add the block to the chain
	sm.bc.Add(block)

	sm.transState = state.Copy()

	fmt.Println("post mining: block=", block)
	fmt.Println("post mining: parent=", parent)

	statelogger.Infof("Imported block #%d (%x...)\n", block.Number, block.Hash()[0:4])

	return nil
}

func (sm *StateManager) ApplyDiff(state *state.State, parent, block *Block) (receipts Receipts, err error) {
	coinbase := state.GetOrNewStateObject(block.Coinbase)

	// Process the transactions on to current block
	receipts, _, _, err = sm.ProcessTransactions(coinbase, state, block, parent, block.Transactions())
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (sm *StateManager) ValidateBlock(block *Block) error {
	parent := sm.bc.GetBlock(block.PrevHash)

	diff := block.Time - parent.Time
	if diff < 0 {
		return ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Time, sm.bc.CurrentBlock.Time)
	}

	return nil
}

func (sm *StateManager) AccumelateRewards(state *state.State, block, parent *Block) error {
	reward := new(big.Int).Set(BlockReward)

	// Get the account associated with the coinbase
	account := state.GetAccount(block.Coinbase)
	// Reward amount of factoid to the coinbase address
	account.AddAmount(reward)

	account.AddAmount(block.TotalFee)

	return nil
}
