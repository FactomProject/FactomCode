// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
    "github.com/FactomProject/simplecoin/block"
	//"fmt"
	"sync"
)

// Simplecoin Chain
type SCChain struct {
	ChainID         *Hash
	
	NextBlock       block.ISCBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

// Simplecoin Block

