// Copyright 2015 FactomProject Authors. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package process

import (
	"time"
)

const numBuckets = 24

var _ = time.Now()

var buckets [numBuckets] map[[32]byte] int64

var lasttime int64 // hours since 1970

func hours(unix int64) int64 {
		return unix/6
}

// Checks if the timestamp is valid.  If the timestamp is too old or 
// too far into the future, then we don't consider it valid.  Or if we
// have seen this hash before, then it is not valid.  To that end,
// this code remembers hashes tested in the past, and rejects the 
// second submission of the same hash.
func IsTSValid(hash[]byte, timestamp int64) bool {

	now := hours(time.Now().Unix())

	// If we have no buckets, or more than 24 hours has passed, 
	// toss all the buckets. We do this by setting lasttime 24 hours
	// in the past.
	if now-lasttime > int64(numBuckets) {
		lasttime = now-int64(numBuckets)
	}
	
	// for every hour that has passed, toss one bucket by shifting
	// them all down a slot, and allocating a new bucket.
	for lasttime < now {
		for i:=0; i<numBuckets-1; i++ {
			buckets[i]=buckets[i+1]
		}
		buckets[numBuckets-1] = make(map[[32]byte] int64,10)
		lasttime++
	}
	
	
	t := hours(timestamp)
	if t > now+int64(numBuckets)/2 || t < now-int64(numBuckets)/2 {
		return false
	}
	
	index := int(t-now + int64(numBuckets)/2)
	
	var h [32]byte 
	copy(h[:],hash)
	
	_, ok := buckets[index][h]
	if ok {
		return false
	}
	
	buckets[index][h]=t
	
	return true	
}