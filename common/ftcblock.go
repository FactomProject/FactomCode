// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"fmt"
	"github.com/FactomProject/factoid/block"
	"github.com/FactomProject/factoid/state/stateinit"
	"sync"
)

var _ = fmt.Println

var FactoidState = stateinit.NewFactoidState("/tmp/factoid_bolt.db")

// factoid Chain
type FctChain struct {
	ChainID *Hash

	NextBlock       block.IFBlock
	NextBlockHeight uint32
	BlockMutex      sync.Mutex
}

var _ Printable = (*FctChain)(nil)

func (e *FctChain) JSONByte() ([]byte, error) {
	return EncodeJSON(e)
}

func (e *FctChain) JSONString() (string, error) {
	return EncodeJSONString(e)
}

func (e *FctChain) JSONBuffer(b *bytes.Buffer) error {
	return EncodeJSONToBuffer(e, b)
}

func (e *FctChain) Spew() string {
	return Spew(e)
}

// factoid Block
