
package ldb

import (
	"github.com/FactomProject/FactomCode/notaryapi"	
	"github.com/conformal/goleveldb/leveldb"
	"errors"
	"log"

)

// ProcessEBlockBatche inserts the EBlock and update all it's ebentries in DB
func (db *LevelDb) ProcessEBlockBatche(eBlockHash *notaryapi.Hash, eblock *notaryapi.Block) error {

	if eblock !=  nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()
		
		if len(eblock.EBEntries) < 1 {
			return errors.New("Empty eblock!")
		}
				
		binaryEblock, err := eblock.MarshalBinary()
		if err != nil{
			return err
		}
		
		// insert the binary entry block
		var key [] byte = []byte{byte(TBL_EB)} 
		key = append (key, eBlockHash.Bytes ...)			
		db.lbatch.Put(key, binaryEblock)
		
		// insert the binary entry block process queue
/*		var qkey [] byte = []byte{byte(TBL_EB_QUEUE)} 
		
		binaryType := make([]byte, 4)
		binary.BigEndian.PutUint32(binaryType, uint32(eblock.Type()))
		key = append(key, binaryType ...) 								// Chain Type (4 bytes)
		
		binaryTimestamp := make([]byte, 8)
		binary.BigEndian.PutUint64(binaryTimestamp, uint64(entry.TimeStamp()))	
		key = append(key, binaryTimestamp ...) 							// Timestamp (8 bytes)		
		qkey = append (qkey, eBlockHash.Bytes ...)			
		db.lbatch.Put(qkey, []byte{byte(STATUS_IN_QUEUE)})
*/
		for i:=0; i< len(eblock.EBEntries); i++  {
			var ebEntry notaryapi.EBEntry = *eblock.EBEntries[i] 
			var ebEntryKey [] byte = []byte{byte(TBL_ENTRY_QUEUE)} 		  			// Table Name (1 bytes)
			ebEntryKey = append(ebEntryKey, *(eblock.Chain.ChainID) ...) 			// Chain id (32 bytes)
			ebEntryKey = append(ebEntryKey, ebEntry.GetBinaryTimeStamp() ...) 			// Timestamp (8 bytes)
			ebEntryKey = append(ebEntryKey, ebEntry.Hash().Bytes ...) 				// Entry Hash (32 bytes)
			db.lbatch.Put(ebEntryKey, []byte{byte(STATUS_PROCESSED)})
			
			log.Println("ebEntry.BinaryTimestamp:%v", ebEntry.GetBinaryTimeStamp())			
			log.Println("put ebEntryKey:%v", ebEntryKey)
			
		}

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			log.Println("batch failed %v\n", err)
			return err
		}

	}
	return nil
}