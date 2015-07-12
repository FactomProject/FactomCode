package factomapi

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/gocoding"
	"net/http"
	"net/url"
	"os"
	"testing"
	//	"encoding/base64"
	//	"io/ioutil"
	//"os"
)

var (
	_    bytes.Buffer
	_, _ = hex.DecodeString("abcd")
	_    gocoding.Marshaller
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
		t.Log("Buy credit request successfully submitted to factomclient.")
	}

}

//need to call commit first because we added payment logic on server
func TestAddChain(t *testing.T) {
	fmt.Println("\nTestAddChain===========================================================================")
	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("Project Gutenberg"))
	bName = append(bName, []byte("5c1d290be98200b29dd1c3fe"))

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

	t.Log("chain:%v", string(buf.Bytes()))
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
		t.Log("Chain successfully submitted to factomclient.")
	}

	// JSON ws test done ----------------------------------------------------------------------------
}

func TestAddEntry(t *testing.T) {
	fmt.Println("\nTestAddEntry===========================================================================")

	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("Project Gutenberg"))
	bName = append(bName, []byte("5c1d290be98200b29dd1c3fe"))

	chain.Name = bName
	chain.GenerateIDFromName()

	file, _ := os.Open("/tmp/gutenberg/entries")
	scanner := bufio.NewScanner(file)
	for n := 0; scanner.Scan(); {
		n++
		e, err := UnmarshalJSON([]byte(scanner.Text()))
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		e.ChainID = *chain.ChainID

		buf := new(bytes.Buffer)
		err = SafeMarshal(buf, e)

		jsonstr := string(buf.Bytes())

		// Post the entry JSON to FactomClient web server
		data := url.Values{}
		data.Set("entry", jsonstr)
		data.Set("format", "json")
		data.Set("password", "opensesame")

		resp, err := http.PostForm("http://localhost:8088/v1/submitentry", data)
		if err != nil {
			t.Errorf("Error:%v", err)
		} else {
			t.Log(n)
		}
		resp.Body.Close()
	}
}
