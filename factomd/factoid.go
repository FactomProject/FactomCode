// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package factomd

// Various validation checks, for Factoid TXs & Blocks.

import (
	"github.com/FactomProject/FactomCode/util"
)

// 1-byte version
func FactoidTx_VersionCheck(version uint8) bool {
	util.Trace()
	return (0 == version)
}

// in reality: 5 bytes
func FactoidTx_LocktimeCheck(locktime uint64) bool {
	util.Trace()
	return (0 == locktime)
}
