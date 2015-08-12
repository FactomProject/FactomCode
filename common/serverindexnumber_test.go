// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package common_test

import (
	"testing"
	"github.com/FactomProject/FactomCode/common"
)

func TestServerIndexMarshalUnmarshal(t *testing.T) {
	si1 := common.NewServerIndexNumber()
	si1.Number = 3
	b, err := si1.MarshalBinary()
	if err != nil {
		t.Error(err)
	}
	if len(b) != 1 {
		t.Error("Invalid byte length")
	}
	if b[0] != 3 {
		t.Error("Invalid byte")
	}
	
	si2 := common.NewServerIndexNumber()
	err = si2.UnmarshalBinary(b)
	if err != nil {
		t.Error(err)
	}
	if si1.Number != si2.Number {
		t.Error("Invalid data unmarshalled")
	}
}
