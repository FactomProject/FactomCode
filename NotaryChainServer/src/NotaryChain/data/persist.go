package data

import (
	"github.com/coopernurse/gorp"
	"database/sql"
	"errors"
)

type plainEntryGORP struct {
	PlainEntry
	EntryID, BlockID uint64
}

type NcDbMap struct {
	gorp.DbMap
}

type typeConverter struct {}

func InitGORP(db *sql.DB) *NcDbMap {
	dbmap := &NcDbMap{gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}}
	
	dbmap.AddTable(Block{}).SetKeys(true, "BlockID")
	dbmap.AddTableWithName(plainEntryGORP{}, "Entry").SetKeys(false, "EntryID", "BlockID")
	dbmap.TypeConverter = typeConverter{}
	
	if err := dbmap.CreateTablesIfNotExists(); err != nil {
		panic(err)
	}
	
	return dbmap
}

func (m *NcDbMap) SaveBlock(block *Block) (err error) {
	if err = m.Insert(block); err != nil { return }
	
	for entryID, entry := range block.Entries {
		if err = m.Insert(&plainEntryGORP{*entry, uint64(entryID), block.BlockID}); err != nil { return }
	}
	
	return
}

func (m *NcDbMap) LoadBlocks() (blocks []*Block, err error) {
	count, err := m.SelectInt("SELECT COUNT(*) FROM Block")
	if err != nil { return }
	
	start, err := m.SelectInt("SELECT MIN(BlockID) FROM Block")
	if err != nil { return }
	
	blocks = make([]*Block, count, 2*count + 1)
	for i := int64(0); i < count; i = i + 1 {
		err = m.SelectOne(blocks[i], "SELECT * FROM Block WHERE BlockID = ?", i + start)
		if err != nil { return }
		if blocks[i] == nil {
			err = errors.New("Blocks in database are non sequential")
			return
		}
	}
	
	return
}

func (c typeConverter) ToDb(val interface{}) (interface{}, error) {
	switch val.(type) {
	case Hash:
		return val.(*Hash).Bytes, nil
		
	default:
		return val, nil
	}
}

func (c typeConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch target.(type) {
	case *Hash:
		return gorp.CustomScanner{new([]byte), target, func(holder, target interface{}) error {
				b, ok := holder.(*[]byte)
				if !ok {
					return errors.New("FromDb: Unable to convert Hash to *[]byte")
				}
				h, ok := target.(*Hash)
				if !ok {
					return errors.New("FromDb: Unable to convert target to *Hash")
				}
				h.Bytes = *b
				return nil
			}}, true
	}
	return gorp.CustomScanner{}, false
}