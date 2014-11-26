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
	c := new(notaryapi.Chain)
	c.Name = name	
	c.GenerateIDFromName()
	return c.ChainID, nil
}

func RevealChain(version uint16, c *notaryapi.Chain, e *notaryapi.Entry) error {
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

func GetDirectoryBloks(fromBlockHeight uint64, toBlockHeight uint64) (fBlocks []notaryapi.FBlock, err error) {
	//needs to be improved ??
	fBlocks, _ = db.FetchAllFBlocks()
	sort.Sort(byBlockID(fBlocks))
	
	if fromBlockHeight > uint64(len(fBlocks)-1) {
		return nil, nil
	} else if toBlockHeight > uint64(len(fBlocks)-1) {
		toBlockHeight = uint64(len(fBlocks)-1)
	}
	
	return fBlocks[fromBlockHeight:toBlockHeight+1], nil
}


func GetDirectoryBlokByHash(fBlockHash *notaryapi.Hash) (fBlock *notaryapi.FBlock, err error) {

	fBlock, err = db.FetchFBlockByHash(fBlockHash)
	
	return fBlock, err
}

func GetDirectoryBlokByHashStr(fBlockHashBase64 string) (fBlock *notaryapi.FBlock, err error) {
	
	bytes, err := base64.StdEncoding.DecodeString(fBlockHashBase64)
	
	
	if err != nil || len(bytes) != notaryapi.HashSize{
		return nil, err
	}
	fBlockHash := new (notaryapi.Hash)
	fBlockHash.Bytes = bytes
	
	
	fBlock, _ = db.FetchFBlockByHash(fBlockHash)
	
	return fBlock, nil
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
type byBlockID []notaryapi.FBlock
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
type byEBlockID []notaryapi.Block
func (f byEBlockID) Len() int { 
  return len(f) 
} 
func (f byEBlockID) Less(i, j int) bool { 
  return f[i].Header.BlockID > f[j].Header.BlockID
} 
func (f byEBlockID) Swap(i, j int) { 
  f[i], f[j] = f[j], f[i] 
} 

