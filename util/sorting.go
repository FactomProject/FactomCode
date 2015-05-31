package util

import (
    "github.com/FactomProject/FactomCode/common"
    "github.com/FactomProject/simplecoin/block"
)

//------------------------------------------------
// DBlock array sorting implementation - accending
type ByDBlockIDAccending []common.DirectoryBlock

func (f ByDBlockIDAccending) Len() int {
	return len(f)
}
func (f ByDBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.BlockHeight < f[j].Header.BlockHeight
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
	return f[i].Header.DBHeight < f[j].Header.DBHeight
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
type BySCBlockIDAccending []block.ISCBlock

func (f BySCBlockIDAccending) Len() int {
    return len(f)
}
func (f BySCBlockIDAccending) Less(i, j int) bool {
    return f[i].GetDBHeight() < f[j].GetDBHeight()
}
func (f BySCBlockIDAccending) Swap(i, j int) {
    f[i], f[j] = f[j], f[i]
}

//------------------------------------------------
// EBlock array sorting implementation - accending
type ByEBlockIDAccending []common.EBlock

func (f ByEBlockIDAccending) Len() int {
	return len(f)
}
func (f ByEBlockIDAccending) Less(i, j int) bool {
	return f[i].Header.EBHeight < f[j].Header.EBHeight
}
func (f ByEBlockIDAccending) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
