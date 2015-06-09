// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
    "github.com/FactomProject/factoid/block"
	//"fmt"
	"sync"
)

// factoid Chain
type SCChain struct {
	ChainID         *Hash
	
	NextBlock       block.IFBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

// factoid Block

