package notaryapi

import (
	"net/http"
	"net/url"
	"fmt"
	"encoding/hex"	

)

var serverAddr = "localhost:8083"	
func CommitChain(name [][]byte) (*Hash, error) {
	c := new(Chain)
	c.Name = name	
	c.GenerateIDFromName()
	return c.ChainID, nil
}

func RevealChain(version uint16, c *Chain, e *Entry) error {
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
func RevealEntry(version uint16, e *Entry) error {
	bEntry,_ := e.MarshalBinary()

	data := url.Values{}
	data.Set("format", "binary")
	data.Set("entry", hex.EncodeToString(bEntry))
	
	
	fmt.Println("Entry extid[0]:%s", string(e.ExtIDs[0]))
		
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)

	return err
}


func SetServerAddr(addr string) error {
	serverAddr = addr
	
	return nil
}


