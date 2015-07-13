package common

import (
	"fmt"
	"math/rand"
	"testing"
)

func mix(v []uint64) {
	for i := 0; i < 100; i++ {
		v1 := rand.Int() % len(v)
		v2 := rand.Int() % len(v)
		t := v[v1]
		v[v1] = v[v2]
		v[v2] = t
	}
}

func TestVarIntLength(t *testing.T) {
    fmt.Println("VarInt")

	if VarIntLength(0) != 1 {
		t.Error("Wrong length for 0")
	}
	if VarIntLength(1) != 1 {
		t.Error("Wrong length for 1")
	}
	if VarIntLength(0xfc) != 1 {
		t.Error("Wrong length for 0xfc")
	}
	if VarIntLength(0xfd) != 3 {
		t.Error("Wrong length for 0xfd")
	}
	if VarIntLength(0xFFFF) != 3 {
		t.Error("Wrong length for 0xFFFF")
	}
	if VarIntLength(0x010000) != 5 {
		t.Error("Wrong length for 0x010000")
	}
	if VarIntLength(0xFFFFFFFF) != 5 {
		t.Error("Wrong length for 0xFFFFFFFF")
	}
	if VarIntLength(0x0100000000) != 9 {
		t.Error("Wrong length for 0x0100000000")
	}
	if VarIntLength(0xFFFFFFFFFF) != 9 {
		t.Error("Wrong length for 0xFFFFFFFFFF")
	}
}
