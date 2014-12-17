package factomapi

import (
	"net/http"
	"net/url"
	"fmt"
	"encoding/hex"	
	"encoding/base64"		
	"sort"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/notaryapi"		
	//"github.com/FactomProject/FactomCode/database/ldb"			

)
//to be improved:
var serverAddr = "localhost:8083"	
var db database.Db // database


func CommitChain(name [][]byte) (*notaryapi.Hash, error) {
	c := new(notaryapi.EChain)
	c.Name = name	
	c.GenerateIDFromName()
	return c.ChainID, nil
}

func RevealChain(version uint16, c *notaryapi.EChain, e *notaryapi.Entry) error {
	bChain,_ := c.MarshalBinary()
	
	data := url.Values {}	
	data.Set("datatype", "chain")
	data.Set("format", "binary")
	data.Set("chain", hex.EncodeToString(bChain))
	
	fmt.Println("chain name[0]:%s", string(c.Name[0]))
	
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)

	return err
}
/*
func CommitEntry(cid *notaryapi.Hash) (*notaryapi.Entry, error) {
	e := new(notaryapi.Entry)
	e.ChainID := cid
	e.ExtHashes := h
	e.Data := data
	
	return e
}
*/
func RevealEntry(version uint16, e *notaryapi.Entry) error {
	bEntry,_ := e.MarshalBinary()

	data := url.Values{}
	data.Set("format", "binary")
	data.Set("entry", hex.EncodeToString(bEntry))
	
	
	fmt.Println("Entry extid[0]:%s", string(e.ExtIDs[0]))
		
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)

	return err
}

func GetDirectoryBloks(fromBlockHeight uint64, toBlockHeight uint64) (dBlocks []notaryapi.DBlock, err error) {
	//needs to be improved ??
	dBlocks, _ = db.FetchAllDBlocks()
	sort.Sort(byBlockID(dBlocks))
	
	if fromBlockHeight > uint64(len(dBlocks)-1) {
		return nil, nil
	} else if toBlockHeight > uint64(len(dBlocks)-1) {
		toBlockHeight = uint64(len(dBlocks)-1)
	}
	
	return dBlocks[fromBlockHeight:toBlockHeight+1], nil
}


func GetDirectoryBlokByHash(dBlockHash *notaryapi.Hash) (dBlock *notaryapi.DBlock, err error) {

	dBlock, err = db.FetchDBlockByHash(dBlockHash)
	
	return dBlock, err
}

func GetDirectoryBlokByHashStr(dBlockHashBase64 string) (dBlock *notaryapi.DBlock, err error) {
	
	bytes, err := base64.StdEncoding.DecodeString(dBlockHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	dBlockHash := new (notaryapi.Hash)
	dBlockHash.Bytes = bytes
	
	
	dBlock, _ = db.FetchDBlockByHash(dBlockHash)
	
	return dBlock, nil
}

func GetEntryBlokByHashStr(eBlockHashBase64 string) (eBlock *notaryapi.EBlock, err error) {
	bytes, err := base64.StdEncoding.DecodeString(eBlockHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	eBlockHash := new (notaryapi.Hash)
	eBlockHash.Bytes = bytes

	return GetEntryBlokByHash(eBlockHash)
}

func GetEntryBlokByHash(eBlockHash *notaryapi.Hash) (eBlock *notaryapi.EBlock, err error) {

	eBlock, err = db.FetchEBlockByHash(eBlockHash)
	
	return eBlock, err
}

func GetEntryByHashStr(entryHashBase64 string) (entry *notaryapi.Entry, err error) {
	bytes, err := base64.StdEncoding.DecodeString(entryHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	entryHash := new (notaryapi.Hash)
	entryHash.Bytes = bytes

	return GetEntryByHash(entryHash)
}

func GetEntryByHash(entrySha *notaryapi.Hash) (entry *notaryapi.Entry, err error) {

	entry, err = db.FetchEntryByHash(entrySha)

	return entry, err
}


// to be removed------------------------------
func SetServerAddr(addr string) error {
	serverAddr = addr
	
	return nil
}

func SetDB(database database.Db) error {
	db = database
	
	return nil
}
//-=-----------------------------------------

// array sorting implementation
type byBlockID []notaryapi.DBlock
func (f byBlockID) Len() int { 
  return len(f) 
} 
func (f byBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID > f[j].Header.BlockID
} 
func (f byBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 

// array sorting implementation
type byEBlockID []notaryapi.EBlock
func (f byEBlockID) Len() int { 
  return len(f) 
} 
func (f byEBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID > f[j].Header.BlockID
} 
func (f byEBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 

