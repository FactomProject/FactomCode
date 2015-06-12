// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
    "github.com/FactomProject/factoid/block"
    "github.com/FactomProject/factoid/state/stateinit"
    "fmt"
	"sync"
)

var _ = fmt.Println

var FactoidState = stateinit.NewFactoidState("/tmp/factoid_bolt.db")

// factoid Chain
type FctChain struct {
	ChainID         *Hash
	
	NextBlock       block.IFBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

// factoid Block

