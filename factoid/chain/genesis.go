package chain

import (
	"math/big"

	"github.com/FactomProject/FactomCode/factoid/crypto"
	"github.com/FactomProject/FactomCode/factoid/util"
)

var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var EmptyShaList = crypto.Sha3Bin(util.Encode([]interface{}{}))

var GenesisHeader = []interface{}{
	// Previous hash (none)
	ZeroHash256,
	// Coinbase
	ZeroHash160,
	// Root state
	"",
	// tx sha
	"",
	// Number
	util.Big0,
	// Block total fee
	util.Big0,
	// Time
	util.Big0,
	// Nonce
	crypto.Sha3Bin(big.NewInt(42).Bytes()),
}

var Genesis = []interface{}{GenesisHeader, []interface{}{}, []interface{}{}}
