package ldb


import (
	"fmt"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid/block"
)

func (db *LevelDb) FetchAllBlockByDBlockHeight(dBlockHeight uint32) (*common.DirectoryBlock, *common.AdminBlock, *common.ECBlock, block.IFBlock, []*common.EBlock, error) {
	dBlockHash, err := db.FetchDBHashByHeight(dBlockHeight)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return db.FetchAllBlockByDBlockHash(dBlockHash)
}


func (db *LevelDb) FetchAllBlockByDBlockHash(dBlockHash *common.Hash) (*common.DirectoryBlock, *common.AdminBlock, *common.ECBlock, block.IFBlock, []*common.EBlock, error) {
	if dBlockHash==nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("dBlockHash not provided")
	}
	dBlock, err := db.FetchDBlockByHash(dBlockHash)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	dbEntries:=dBlock.DBEntries
	if len(dbEntries)<3 {
		return nil, nil, nil, nil, nil, fmt.Errorf("Malformed dBlock")
	}
	aBlock, err:=db.FetchABlockByHash(dbEntries[0].KeyMR)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	ecBlock, err:=db.FetchECBlockByHash(dbEntries[1].KeyMR)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	fBlock, err:=db.FetchFBlockByHash(dbEntries[2].KeyMR)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	dbEntries = dbEntries[3:]
	eBlocks:=make([]*common.EBlock, len(dbEntries))
	for i, entry:=range(dbEntries) {
		eBlock, err:=db.FetchEBlockByMR(entry.KeyMR)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		eBlocks[i] = eBlock
	}

	return dBlock, aBlock, ecBlock, fBlock, eBlocks, nil
}