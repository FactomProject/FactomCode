package factomclient

import (
	"net/http"
	"net/url"
	"encoding/hex"	

	"github.com/FactomProject/FactomCode/core"
)

//for now just set the server statically
var server string = "http://demo.factom.org/v1"

func CreateChain(name [][]byte) (*core.Chain, error) {
	c := new(core.Chain)
	c.Name := name	
	c.GenerateID()
	
	data := url.Values{}
	data.Set("datatype", "createchain")
	data.Set("format", "binary")
	data.Set("chainid", c.ChainID)
	
	resp, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	err := checkResponse(resp)
	if err != nil {
		return err
	}
	
	return c
}

func CreateEntry(cid *Hash) (*core.Entry, error) {
	e := new(core.Entry)
	e.ChainID := cid
	
	return e
}

func CommitEntry(e *core.Entry) error {
	data := url.Values{}
	data.Set("method", "commitentry")
	data.Set("format", "binary")
	data.Set("entryhash", e.Hash())
	
	resp, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

func RevealEntry(e *core.Entry) error {
	data := url.Values{}
	data.Set("datatype", "revealentry")
	data.Set("format", "binary")
	data.Set("entry", hex.EncodeToString(e.MarshalBinary()))
	
	resp, err := http.PostForm(server, data)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

func checkResponse(*http.Response) error {
	// return an error if the http.Response conains information about a factom failure
	return nil
}

