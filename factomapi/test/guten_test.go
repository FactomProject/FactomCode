package factomapi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/gocoding"
	"net/http"
	"net/url"
	"testing"
	//	"encoding/base64"
	//	"io/ioutil"
	//"os"
)

var (
	_ bytes.Buffer
	_, _ = hex.DecodeString("abcd")
	_ gocoding.Marshaller
)

func TestBuyCreditWallet(t *testing.T) {
	fmt.Println("\nTestBuyCreditWallet===========================================================================")
	// Post the request to FactomClient web server	---------------------------------------
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
		t.Errorf("Error:%v", err)
	} else {
		fmt.Println("Buy credit request successfully submitted to factomclient.")
	}

}

//need to call commit first because we added payment logic on server
func TestAddChain(t *testing.T) {
	fmt.Println("\nTestAddChain===========================================================================")
	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("5c1d290be98200b29dd1c3fe"))
	bName = append(bName, []byte("Project Gutenberg"))

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
	err := SafeMarshal(buf, chain)

	fmt.Println("chain:%v", string(buf.Bytes()))
	jsonstr := string(buf.Bytes())

	// Post the chain JSON to FactomClient web server	------------------------
	data := url.Values{}
	data.Set("chain", jsonstr)
	data.Set("format", "json")
	data.Set("password", "opensesame")

	_, err = http.PostForm("http://localhost:8088/v1/submitchain", data)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else {
		fmt.Println("Chain successfully submitted to factomclient.")
	}

	// JSON ws test done ----------------------------------------------------------------------------
}

func TestAddEntry(t *testing.T) {
	fmt.Println("\nTestAddEntry===========================================================================")

	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("5c1d290be98200b29dd1c3fe"))
	bName = append(bName, []byte("Project Gutenberg"))

	chain.Name = bName
	chain.GenerateIDFromName()

	file, _ := os.Open("/tmp/gutenberg/entries")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		e, err := UnmarshalJSON([]byte(scanner.Text()))
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		entry.ChainID = *chain.ChainID
	
		buf := new(bytes.Buffer)
		err := SafeMarshal(buf, entry)
	
		jsonstr := string(buf.Bytes())
	
		// Post the entry JSON to FactomClient web server
		data := url.Values{}
		data.Set("entry", jsonstr)
		data.Set("format", "json")
		data.Set("password", "opensesame")
	
		_, err = http.PostForm("http://localhost:8088/v1/submitentry", data)
		if err != nil {
			t.Errorf("Error:%v", err)
		} else {
			fmt.Println("Entry successfully submitted to factomclient.")
		}
	}
}
