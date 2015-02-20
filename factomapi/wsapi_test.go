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

/*
func TestBuyCredit(t *testing.T) {
	fmt.Println("\nTestBuyCredit===========================================================================")
	// Post the request to FactomClient web server	---------------------------------------
	data := url.Values{}
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)
	data.Set("to", hex.EncodeToString(pubKey.Bytes))
	data.Set("value", "1.123456789")
	data.Set("password", "opensesame")

	_, err := http.PostForm("http://localhost:8088/v1/buycredit", data)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Buy credit request successfully submitted to factomclient.")
	}
}
*/

func TestBuyCreditWallet(t *testing.T) {
	fmt.Println("\nTestBuyCreditWallet===========================================================================")
	// Post the request to FactomClient web server	---------------------------------------
	data := url.Values{}
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new(notaryapi.Hash)
	pubKey.SetBytes(barray)
	data.Set("to", "wallet")
	data.Set("value", "11.123456789")
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
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping3"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new(notaryapi.Entry)
	entry.ChainID = *chain.ChainID
	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("First entry for chain:\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\"Rules:\"asl;djfasldkfjasldfjlksouiewopurw\"")

	chain.FirstEntry = entry

	buf := new(bytes.Buffer)
	err := SafeMarshal(buf, chain)

	fmt.Println("chain:%v", string(buf.Bytes()))
	jsonstr := string(buf.Bytes())

	// Unmarshal the json string locally to compare
	//	jsonstr := "{\"Name\":[\"bXlDb21wYW55\",\"Ym9va2tlZXBpbmc=\"],\"FirstEntry\":{\"ExtIDs\":[\"MTAwMQ==\",\"NTcwYjllM2ZiMmY1YWU4MjM2ODVlYjQ0MjJkNGZkODNmM2YwZDllN2NlMDdkOTg4YmQxN2U2NjUzOTQ2NjhjNg==\",\"bXZSSnFNVE1mclkzS3RIMkE0cWRQZnEzUTZMNEt3OUNrNA==\"],\"Data\":\"Rmlyc3QgZW50cnkgZm9yIGNoYWluOiIyRnJnRDIrdlBQM3l6NXpMVmFFNVRjMlZpVnY5ZndaZVIzL2FkeklUakpjPSJSdWxlczoiYXNsO2RqZmFzbGRrZmphc2xkZmpsa3NvdWlld29wdXJ3Ig==\"}}"
	//fmt.Println(jsonstr)

	// Post the chain JSON to FactomClient web server	---------------------------------------
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

	chain3 := new(notaryapi.EChain)
	reader := gocoding.ReadBytes([]byte(jsonstr))
	err = SafeUnmarshal(reader, chain3)

	fmt.Println("chainid:%v", hex.EncodeToString(chain3.ChainID.Bytes))
	fmt.Println("name0:%v", string(chain3.Name[0]))
	fmt.Println("entrydata:%v", string(chain3.FirstEntry.Data))

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}

func TestAddEntry(t *testing.T) {
	fmt.Println("\nTestAddEntry===========================================================================")

	chain := new(notaryapi.EChain)
	bName := make([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))
	bName = append(bName, []byte("bookkeeping3"))

	chain.Name = bName
	chain.GenerateIDFromName()

	entry := new(notaryapi.Entry)
	entry.ChainID = *chain.ChainID
	entry.ExtIDs = make([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw\"")

	buf := new(bytes.Buffer)
	err := SafeMarshal(buf, entry)

	fmt.Println("entry:%v", string(buf.Bytes()))
	jsonstr := string(buf.Bytes())

	//jsonstr := "{\"ChainID\":\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\",\"ExtIDs\":[\"MTAwMQ==\",\"NTcwYjllM2ZiMmY1YWU4MjM2ODVlYjQ0MjJkNGZkODNmM2YwZDllN2NlMDdkOTg4YmQxN2U2NjUzOTQ2NjhjNg==\",\"bXZSSnFNVE1mclkzS3RIMkE0cWRQZnEzUTZMNEt3OUNrNA==\"],\"Data\":\"RW50cnkgZGF0YTogYXNsO2RqZmFzbGRrZmphc2xkZmpsa3NvdWlld29wdXJ3Ig==\"}"
	//fmt.Println(jsonstr)

	// Post the entry JSON to FactomClient web server	---------------------------------------
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
	// JSON ws test done ----------------------------------------------------------------------------

	entry2 := new(notaryapi.Entry)
	reader := gocoding.ReadBytes([]byte(jsonstr))
	err = SafeUnmarshal(reader, entry2)

	//	fmt.Println("chainid:%v", base64.URLEncoding.EncodeToString(entry2.ChainID.Bytes))
	fmt.Println("ExtIDs0:%v", string(entry2.ExtIDs[0]))
	fmt.Println("entrydata:%v", string(entry2.Data))

	if err != nil {
		t.Errorf("Error:%v", err)
	}
}

/*
func TestBuyCredit(t *testing.T) {
	fmt.Println("\nTestBuyCredit===========================================================================")
	// Post the request to FactomClient web server	---------------------------------------
	data := url.Values{}
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)
	data.Set("to", base64.URLEncoding.EncodeToString(pubKey.Bytes))
	data.Set("value", "1.123456789")
	data.Set("password", "opensesame")

	_, err := http.PostForm("http://localhost:8088/v1/buycredit", data)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Buy credit request successfully submitted to factomclient.")
	}
}



func TestGetCreditBalance(t *testing.T) {
	fmt.Println("\nTestGetCreditBalance===========================================================================")
	// Post the request to FactomClient web server	---------------------------------------
	data := url.Values{}
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)
	data.Set("pubkey", base64.URLEncoding.EncodeToString(pubKey.Bytes))
	data.Set("password", "opensesame")

	resp, err := http.PostForm("http://localhost:8088/v1/creditbalance", data)
	contents, err := ioutil.ReadAll(resp.Body)

	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Get Credit Balance request successfully submitted to factomclient.")
	}
}

func TestGetCreditWalletBalance(t *testing.T) {
	fmt.Println("\nTestGetCreditWalletBalance===========================================================================")
	// Post the request to FactomClient web server	---------------------------------------
	data := url.Values{}
	barray := (make([]byte, 32))
	barray[0] = 2
	pubKey := new (notaryapi.Hash)
	pubKey.SetBytes(barray)
	data.Set("pubkey", "wallet")
	data.Set("password", "opensesame")

	resp, err := http.PostForm("http://localhost:8088/v1/creditbalance", data)
	contents, err := ioutil.ReadAll(resp.Body)

	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Get Credit Balance request successfully submitted to factomclient.")
	}
}


func TestGetDBlocksByRange(t *testing.T) {
	fmt.Println("\nTestGetDBlocksByRange===========================================================================")

	// Send request to FactomClient web server	--------------------------------------
	resp, err := http.Get("http://localhost:8088/v1/dblocksbyrange/0/5")
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Request dblocksbyrange successfully submitted to factomclient.")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("after contents: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}


func TestGetDBlockByHash(t *testing.T) {
	fmt.Println("\nTestGetDBlockByHash===========================================================================")

	// Send request to FactomClient web server	---------------------------------------
	// Copy it from explorer
//	bytes, _ := hex.DecodeString("e6354e9cb2d1e14f18f61c002f02d8ab978ccf56ad716f9f8ad6ce2a807d2614")

//	base64str := base64.URLEncoding.EncodeToString(bytes)

	// copy "prevBlockHash" from the output of the TestGetDBlocksByRange
	var base64str = "oPpQqM+k8ur/ilakzyb+NYt1WJ1F9Zf9kiP9fcdT9og="

	bytes, _ := base64.StdEncoding.DecodeString(base64str)

	base64str = base64.URLEncoding.EncodeToString(bytes)

	resp, err := http.Get("http://localhost:8088/v1/dblock/"+base64str)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Request TestGetDBlockByHash successfully submitted to factomclient.")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("after contents: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}

func TestGetEBlockByMR(t *testing.T) {
	fmt.Println("\nTestGetEBlockByMR===========================================================================")
	// Send request to FactomClient web server	---------------------------------------
	// Copy it from explorer
//	bytes, _ := hex.DecodeString("e91657e97c3c0854707dc7af808936c57a75de0f4d2beb353caeb9fb1aadbdd4")

//	base64str := base64.URLEncoding.EncodeToString(bytes)

	// copy "MerkleRoot" from the output of the TestGetDBlocksByRange
	var base64str = "jNWm5hEt9knuNnytr3sQbikBd7mZJVmdl9GyG0oAuHI="
	bytes, _ := base64.StdEncoding.DecodeString(base64str)

	base64str = base64.URLEncoding.EncodeToString(bytes)

	resp, err := http.Get("http://localhost:8088/v1/eblockbymr/"+base64str)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Request TestGetEBlockByMR successfully submitted to factomclient.")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("after contents: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}

func TestGetEBlockByHash(t *testing.T) {
	fmt.Println("\nTestGetEBlockByHash===========================================================================")
	// Send request to FactomClient web server	---------------------------------------
	// Copy it from explorer
//	bytes, _ := hex.DecodeString("e91657e97c3c0854707dc7af808936c57a75de0f4d2beb353caeb9fb1aadbdd4")

//	base64str := base64.StdEncoding.EncodeToString(bytes)

	// copy "prevBlockHash" from the output of the TestGetDBlocksByRange
	var base64str = "7kqwJnHZLhvE3jPdhBSerAYb+0WZ8RIz2NFRoUDrfEY="
	bytes, _ := base64.StdEncoding.DecodeString(base64str)

	base64str = base64.URLEncoding.EncodeToString(bytes)

	resp, err := http.Get("http://localhost:8088/v1/eblock/"+base64str)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Request TestGetEBlockByHash successfully submitted to factomclient.")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("after contents: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}


func TestGetEntryByHash(t *testing.T) {
	fmt.Println("\nTestGetEntryByHash===========================================================================")
	// Send request to FactomClient web server	---------------------------------------
	// Copy it from explorer
	//bytes, _ := hex.DecodeString("392a14b90ad64a340514b869795a58ec5461b0cc2d31a735e388b62d9f7a77ee")

	//base64str := base64.StdEncoding.EncodeToString(bytes)

	//copy it from the output of the tests above
	var base64str = "x6vjCOeND2NLtkDK16cc/VU1hKqGol/GYsygUQTpjXU="
	bytes, _ := base64.StdEncoding.DecodeString(base64str)

	base64str = base64.URLEncoding.EncodeToString(bytes)


	resp, err := http.Get("http://localhost:8088/v1/entry/"+base64str)
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Request TestGetEntryByHash successfully submitted to factomclient.")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("after contents: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}
*/
