package restapi

import (
	"github.com/FactomProject/btcd/wire"
	"sync"
	"time"
)

// ftmMemPool is used as a source of factom transactions
// (CommitChain, RevealChain, CommitEntry, RevealEntry)
type ftmMemPool struct {
	sync.RWMutex
	pool        map[wire.ShaHash]wire.Message
	orphans     map[wire.ShaHash]wire.Message
	lastUpdated time.Time // last time pool was updated
}

// Add a factom message to the orphan pool
func (mp *ftmMemPool) init_ftmMemPool() error {

	mp.pool = make(map[wire.ShaHash]wire.Message)
	mp.orphans = make(map[wire.ShaHash]wire.Message)

	return nil
}

// Add a factom message to the  Mem pool
func (mp *ftmMemPool) addMsg(msg wire.Message, hash *wire.ShaHash) error {

	mp.pool[*hash] = msg

	return nil
}

// Add a factom message to the orphan pool
func (mp *ftmMemPool) addOrphanMsg(msg wire.Message, hash *wire.ShaHash) error {

	mp.orphans[*hash] = msg

	return nil
}
