package factomclient

import (
	"net/http"
	"net/url"
	"fmt"
	"encoding/hex"	
	"github.com/FactomProject/FactomCode/notaryapi"
)

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
	
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)

	return err
}
/*
func CreateEntry(cid *notaryapi.Hash) (*notaryapi.Entry, error) {
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
	
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	_, err := http.PostForm(server, data)

	return err
}


func SetServerAddr(addr string) error {
	serverAddr = addr
	
	return nil
}
