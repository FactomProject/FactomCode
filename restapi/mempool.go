package restapi

import (
	"sync"
	"time"
	"github.com/FactomProject/btcd/wire"	
)

// ftmMemPool is used as a source of factom transactions 
// (CommitChain, RevealChain, CommitEntry, RevealEntry)
type ftmMemPool struct {
	sync.RWMutex
	pool          map[wire.ShaHash]wire.Message
	orphans       map[wire.ShaHash]wire.Message
	lastUpdated   time.Time // last time pool was updated
}
