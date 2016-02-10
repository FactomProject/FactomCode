package util

import (
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid/block"
)

//------------------------------------------------
// DBlock array sorting implementation - accending
type ByDBlockIDAccending []common.DirectoryBlock

func (f ByDBlockIDAccending) Len() int {
	return len(f)
}
func (f ByDBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.DBHeight < f[j].Header.DBHeight
}
func (f ByDBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// CBlock array sorting implementation - accending
type ByECBlockIDAccending []common.ECBlock

func (f ByECBlockIDAccending) Len() int {
	return len(f)
}
func (f ByECBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.EBHeight < f[j].Header.EBHeight
}
func (f ByECBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// ABlock array sorting implementation - accending
type ByABlockIDAccending []common.AdminBlock

func (f ByABlockIDAccending) Len() int {
	return len(f)
}
func (f ByABlockIDAccending) Less(i, j int) bool {
	return f[i].Header.DBHeight < f[j].Header.DBHeight
}
func (f ByABlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// ABlock array sorting implementation - accending
type ByFBlockIDAccending []block.IFBlock

func (f ByFBlockIDAccending) Len() int {
	return len(f)
}
func (f ByFBlockIDAccending) Less(i, j int) bool {
	return f[i].GetDBHeight() < f[j].GetDBHeight()
}
func (f ByFBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// EBlock array sorting implementation - accending
type ByEBlockIDAccending []common.EBlock

func (f ByEBlockIDAccending) Len() int {
	return len(f)
}
func (f ByEBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.EBSequence < f[j].Header.EBSequence
}
func (f ByEBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
