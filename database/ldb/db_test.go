package ldb

import (
	"bytes"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/FactomCode/database"
	"os"
	"testing"
)

var dbFilename string = "levelTest.db"

func TestFetchDBlockByHeight(t *testing.T) {
	db, err := newDB()
	if err != nil {
		t.Errorf("%v", err)
	}
	defer CleanupTest(t, db)

	dchain := common.NewDChain()
	blockCount := 10
	dblocks := make([]*common.DirectoryBlock, 0, blockCount)
	var prev *common.DirectoryBlock = nil
	for i := 0; i < blockCount; i++ {
		dchain.NextDBHeight = uint32(i)
		prev, err = common.CreateDBlock(dchain, prev, 10)
		if err != nil {
			t.Errorf("TestFetchDBlockByHeight error saving block %v - %v", i, err)
		}
		dblocks = append(dblocks, prev)
		err = db.ProcessDBlockBatch(prev)
		if err != nil {
			t.Errorf("TestFetchDBlockByHeight error saving block %v - %v", i, err)
		}
	}

	for i, v := range dblocks {
		block, err := db.FetchDBlockByHeight(uint32(i))
		if err != nil {
			t.Errorf("TestFetchDBlockByHeight error loading block %v - %v", i, err)
		}
		hex1, err := v.MarshalBinary()
		if err != nil {
			t.Errorf("TestFetchDBlockByHeight error marshalling block %v - %v", i, err)
		}
		hex2, err := block.MarshalBinary()
		if err != nil {
			t.Errorf("TestFetchDBlockByHeight error marshalling block %v - %v", i, err)
		}
		if bytes.Compare(hex1, hex2) != 0 {
			t.Errorf("TestFetchDBlockByHeight blocks at height %v do not match", i)
		}
	}
	//FetchDBlockByHeight
}

func newDB() (database.Db, error) {
	return OpenLevelDB(dbFilename, true)
}

func CleanupTest(t *testing.T, b database.Db) {
	err := b.Close()
	if err != nil {
		t.Errorf("%v", err)
	}
	err = os.RemoveAll(dbFilename)
	if err != nil {
		t.Errorf("%v", err)
	}
	err = os.RemoveAll(dbFilename + ".ver")
	if err != nil {
		t.Errorf("%v", err)
	}
}
