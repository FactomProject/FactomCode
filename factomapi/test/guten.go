package main

import (
	"bytes"
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/gocoding"
)

var (
	_    bytes.Buffer
	_    = fmt.Sprint("testing")
	_    gocoding.Marshaller
	_, _ = hex.DecodeString("abcd")
)

func UnmarshalJSON(b []byte) (*notaryapi.Entry, error) {
	type entry struct {
		ChainID string
		ExtIDs  []string
		Data    string
	}
	
	var je entry
	e := new(notaryapi.Entry)
	
	err := json.Unmarshal(b, &je)
	if err != nil {
		return nil, err
	}
	
	bytes, err := hex.DecodeString(je.ChainID)
	if err != nil {
		return nil, err
	}
	e.ChainID.Bytes = bytes
	
	for _, v := range je.ExtIDs {
		e.ExtIDs = append(e.ExtIDs, []byte(v))
	}
	bytes, err = hex.DecodeString(je.Data)
	if err != nil {
		return nil, err
	}
	e.Data = bytes
	
	return e, nil
}

func BuyCreditWallet() {
	data := url.Values{}
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(barray)
	data.Set("to", "wallet")
	data.Set("value", "111111")
	data.Set("password", "opensesame")

	_, err := http.PostForm("http://localhost:8088/v1/buycredit", data)
	if err != nil {
		log.Println(err)
	}
}

//need to call commit first because we added payment logic on server
func AddChain() {
	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("Project Guttenberg"))
	bName = append(bName, []byte("3e0435d280ef6e1"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new(notaryapi.Entry)
	entry.ChainID = *chain.ChainID
	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("Project Gutenberg"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("books"))
	entry.Data = []byte("Project Gutenberg Books. Each entry corresponds to a book in the Project Gutenberg library, and contains a hash of the file. Each Entry should have External IDs for the author and for the title of the work.")

	chain.FirstEntry = entry

	buf := new(bytes.Buffer)
	err := factomapi.SafeMarshal(buf, chain)

	log.Println("chain:", string(buf.Bytes()))
	jsonstr := string(buf.Bytes())

	// Post the chain JSON to FactomClient web server	------------------------
	data := url.Values{}
	data.Set("chain", jsonstr)
	data.Set("format", "json")
	data.Set("password", "opensesame")

	_, err = http.PostForm("http://localhost:8088/v1/submitchain", data)
	if err != nil {
		log.Println("Error:", err)
	} else {
		log.Println("Chain successfully submitted to factomclient.")
	}
}

func AddEntry() {
	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("Project Guttenberg"))
	bName = append(bName, []byte("3e0435d280ef6e1"))

	chain.Name = bName
	chain.GenerateIDFromName()

	file, err := os.Open("/tmp/gutenberg/entries")
	if err != nil {
		log.Fatal("could not find entries file")
	}
	scanner := bufio.NewScanner(file)
	line := make(chan string, 25)
	go func() {
		for scanner.Scan() {
			line <- scanner.Text()
		}
	}()
	for v := range line {
		go func() {
			e, err := UnmarshalJSON([]byte(v))
			if err != nil {
				log.Println("Error:", err)
			}
			e.ChainID = *chain.ChainID
		
			buf := new(bytes.Buffer)
			err = factomapi.SafeMarshal(buf, e)
		
			jsonstr := string(buf.Bytes())
		
			// Post the entry JSON to FactomClient web server
			data := url.Values{}
			data.Set("entry", jsonstr)
			data.Set("format", "json")
			data.Set("password", "opensesame")
		
			resp, err := http.PostForm("http://localhost:8088/v1/submitentry", data)
			if err != nil {
				log.Println("Error:", err)
			} else {
				log.Println("Entry successfully submitted to factomclient.")
			}
			resp.Body.Close()
		}()
	}
}

func main() {
	BuyCreditWallet()
	AddChain()
	AddEntry()
}