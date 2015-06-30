// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package factoid

// Various validation checks, for Factoid TXs & Blocks.

import (
	"fmt"
	"github.com/FactomProject/FactomCode/util"
)

// 1-byte version
func FactoidTx_VersionCheck(version uint8) bool {
	//	util.Trace(fmt.Sprintf("version being checked: %d", version))
	return (0 == version)
}

// in reality: 5 bytes
func FactoidTx_LocktimeCheck(locktime int64) bool {
	util.Trace(fmt.Sprintf("locktime being checked: 0x%X", locktime))
	return (0 == locktime)
}

// 1-byte RCD version
func FactoidTx_RCDVersionCheck(version uint8) bool {
	util.Trace()
	return (0 == version)
}

// 1-byte RCD type
func FactoidTx_RCDTypeCheck(rcdtype uint8) bool {
	util.Trace()
	return (0 == rcdtype)
}
