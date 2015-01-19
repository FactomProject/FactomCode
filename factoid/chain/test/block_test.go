package chain

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/FactomProject/FactomCode/factoid/chain"
	"github.com/FactomProject/FactomCode/factoid/crypto"
	//	"github.com/FactomProject/FactomCode/factoid/db"
	"github.com/FactomProject/FactomCode/factoid/util"
)

func TestTransaction(t *testing.T) {
	km := crypto.NewFileKeyManager("/tmp/factomkey")

	km.Init("Bo", 0, false)
	bo_addr := km.Address()
	bo_priv := km.PrivateKey()
	fmt.Println("Bo.addr=", util.Bytes2Hex(bo_addr))

	km.Init("Jack", 0, false)
	jack_addr := km.Address()
	jack_priv := km.PrivateKey()
	fmt.Println("Jack.addr=", util.Bytes2Hex(jack_addr))

	km.Init("Protocol", 0, false)
	protocol := km.Address()
	proto_priv := km.PrivateKey()
	fmt.Println("protocol=", util.Bytes2Hex(protocol))

	//	bc := chain.NewBlockChain()

	//	genesis := bc.Genesis()
	//	fmt.Println("genesis=", genesis)

	sm := chain.NewStateManager() //bc)

	fmt.Println("send.balance=", sm.CurrentState().GetOrNewStateObject(protocol).Balance)
	fmt.Println("rec.balance=", sm.CurrentState().GetOrNewStateObject(bo_addr).Balance)
	fmt.Println("coinbase.balance=", sm.CurrentState().GetOrNewStateObject(jack_addr).Balance)

	block := chain.FChain.NewBlock(jack_addr) //coinbase

	tx := chain.NewTransaction(bo_addr, big.NewInt(100), big.NewInt(1))
	tx.Sign(proto_priv)
	//	fmt.Println(tx.String())
	block.AddTransaction(tx)

	tx = chain.NewTransaction(jack_addr, big.NewInt(100), big.NewInt(1))
	tx.Sign(bo_priv)
	block.AddTransaction(tx)

	tx = chain.NewTransaction(protocol, big.NewInt(100), big.NewInt(1))
	tx.Sign(jack_priv)
	block.AddTransaction(tx)

	tx = chain.NewTransaction(bo_addr, big.NewInt(2), big.NewInt(1))
	tx.Sign(jack_priv)
	block.AddTransaction(tx)

	err := sm.Process(block)
	//	sm.MineNewBlock(block)
	if err != nil {
		fmt.Println("sm.process: ", err.Error())
	}

	fmt.Println("send.balance=", sm.CurrentState().GetOrNewStateObject(protocol).Balance)
	fmt.Println("rec.balance=", sm.CurrentState().GetOrNewStateObject(bo_addr).Balance)
	fmt.Println("coinbase.balance=", sm.CurrentState().GetOrNewStateObject(jack_addr).Balance)
}

/*
func (sm *StateManager) MineNewBlock(block *Block) {
	// Sort the transactions by nonce in case of odd network propagation
	sort.Sort(TxByNonce{block.Transactions()})

	// Accumulate all valid transactions and apply them to the new state
	// Error may be ignored. It's not important during mining
//	parent := sm.BlockChain().GetBlock(block.PrevHash)
	coinbase := block.State().GetOrNewStateObject(block.Coinbase)
	receipts, txs, _, err := sm.ProcessTransactions(coinbase, block.State(), block, block, block.Transactions())
	if err != nil {
		statelogger.Debugln(err)
	}
	//suppose everything goes right
	//self.txs = append(txs, unhandledTxs...)
	if len(txs) != len(block.Transactions()) {
		fmt.Printf("only %d out of %d transactions being mined\n", len(txs), len(block.Transactions()))
	}
	block.SetTxHash(receipts)

	// Set the transactions to the block so the new SHA3 can be calculated
	block.SetReceipts(receipts, txs)

	// Accumulate the rewards included for this block
//	sm.AccumelateRewards(block.State(), block, parent)

	block.State().Update()

	statelogger.Infof("Mining on block. Includes %v transactions", len(block.Transactions()))

	err = sm.Process(block)
	if err != nil {
		statelogger.Infoln(err)
	}
}
*/
