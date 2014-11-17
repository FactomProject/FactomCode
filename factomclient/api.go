package factomclient

import (
	"net/http"
	"net/url"
	"encoding/hex"	

	"github.com/FactomProject/FactomCode/noteryapi"
)

//for now just set the server statically
var server string = "http://demo.factom.org/v1"

func CreateChain(name [][]byte) (*Hash, error) {
	c := new(notaryapi.Chain)
	c.Name := name	
	c.GenerateIDFromName()
	return c.ChainID, nil
}

func SubmitChain(c *notaryapi.Chain) error {
	bin := c.MarshalBinary()
	
	data := url.Values	
	data.Set("datatype", "chain")
	data.Set("format", "binary")
	data.Set("chain", hex.EncodeToString(&bin))
	
	resp, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
}

func CreateEntry(cid *Hash) (*notaryapi.Entry, error) {
	e := new(notaryapi.Entry)
	e.ChainID := cid
	e.ExtHashes := h
	e.Data := data
	
	return e
}

func SubmitEntry(e *noteryapi.Entry) error {
	bin := e.MarshalBinary()

	data := url.Values
	data.Set("format", "binary")
	data.Set("entry", hex.EncodeToString(&bin))

	resp, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
}
