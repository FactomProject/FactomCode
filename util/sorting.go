package util

import (
	"github.com/FactomProject/FactomCode/notaryapi"
)

//------------------------------------------------
// DBlock array sorting implementation - accending
type ByDBlockIDAccending []notaryapi.DBlock

func (f ByDBlockIDAccending) Len() int {
	return len(f)
}
func (f ByDBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.BlockID < f[j].Header.BlockID
}
func (f ByDBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// CBlock array sorting implementation - accending
type ByCBlockIDAccending []notaryapi.CBlock

func (f ByCBlockIDAccending) Len() int {
	return len(f)
}
func (f ByCBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.BlockID < f[j].Header.BlockID
}
func (f ByCBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// EBlock array sorting implementation - accending
type ByEBlockIDAccending []notaryapi.EBlock

func (f ByEBlockIDAccending) Len() int {
	return len(f)
}
func (f ByEBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.BlockID < f[j].Header.BlockID
}
func (f ByEBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
