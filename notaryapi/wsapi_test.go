package notaryapi

import (
	"testing"
	"bytes"	
	"net/http"
	"net/url"
	"fmt"
//	"encoding/hex"	
//	"encoding/json"	
	"github.com/firelizzard18/gocoding"	
	"encoding/base64"
	
)

func TestAddChain(t *testing.T) {

	chain := new (Chain)
	bName := make ([][]byte, 0, 5)
	bName = append(bName, []byte("myCompany"))	
	bName = append(bName, []byte("bookkeeping"))		
	
	chain.Name = bName
	chain.GenerateIDFromName()
	
	entry := new (Entry)
	entry.ExtIDs = make ([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))	
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))	
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("First entry for chain:\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\"Rules:\"asl;djfasldkfjasldfjlksouiewopurw\"")
	
	chain.FirstEntry = entry
	
	
	buf := new(bytes.Buffer)
	err := SafeMarshal(buf, chain)
	
	fmt.Println("chain:%v", string(buf.Bytes()))
	


	// Unmarshal the json string locally to compare
	jsonstr := "{\"ChainID\":\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\",\"Name\":[\"bXlDb21wYW55\",\"Ym9va2tlZXBpbmc=\"],\"FirstEntry\":{\"ExtIDs\":[\"MTAwMQ==\",\"NTcwYjllM2ZiMmY1YWU4MjM2ODVlYjQ0MjJkNGZkODNmM2YwZDllN2NlMDdkOTg4YmQxN2U2NjUzOTQ2NjhjNg==\",\"bXZSSnFNVE1mclkzS3RIMkE0cWRQZnEzUTZMNEt3OUNrNA==\"],\"Data\":\"Rmlyc3QgZW50cnkgZm9yIGNoYWluOiIyRnJnRDIrdlBQM3l6NXpMVmFFNVRjMlZpVnY5ZndaZVIzL2FkeklUakpjPSJSdWxlczoiYXNsO2RqZmFzbGRrZmphc2xkZmpsa3NvdWlld29wdXJ3Ig==\"}}"
	fmt.Println(jsonstr)

	// Post the chain JSON to FactomClient web server	---------------------------------------
	data := url.Values{}
	data.Set("chain", jsonstr)
	data.Set("password", "opensesame")
	
	_, err = http.PostForm("http://localhost:8088/v1/addchain", data)	
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Chain successfully submitted to factomclient.")
	}			
	// JSON ws test done ----------------------------------------------------------------------------

	chain3 := new (Chain)
	reader := gocoding.ReadBytes([]byte(jsonstr))
	err = SafeUnmarshal(reader, chain3)

	fmt.Println("chainid:%v", base64.StdEncoding.EncodeToString(chain3.ChainID.Bytes))
	fmt.Println("name0:%v", string(chain3.Name[0]))	
	fmt.Println("entrydata:%v", string(chain3.FirstEntry.Data))	
		
	if err != nil {
		t.Errorf("Error:%v", err)
	}
} 

func TestAddEntry(t *testing.T) {

	entry := new (Entry)
	entry.ExtIDs = make ([][]byte, 0, 5)
	entry.ExtIDs = append(entry.ExtIDs, []byte("1001"))	
	entry.ExtIDs = append(entry.ExtIDs, []byte("570b9e3fb2f5ae823685eb4422d4fd83f3f0d9e7ce07d988bd17e665394668c6"))	
	entry.ExtIDs = append(entry.ExtIDs, []byte("mvRJqMTMfrY3KtH2A4qdPfq3Q6L4Kw9Ck4"))
	entry.Data = []byte("Entry data: asl;djfasldkfjasldfjlksouiewopurw\"")
	
	buf := new(bytes.Buffer)
	err := SafeMarshal(buf, entry)
	
	fmt.Println("entry:%v", string(buf.Bytes()))

	jsonstr := "{\"ChainID\":\"2FrgD2+vPP3yz5zLVaE5Tc2ViVv9fwZeR3/adzITjJc=\",\"ExtIDs\":[\"MTAwMQ==\",\"NTcwYjllM2ZiMmY1YWU4MjM2ODVlYjQ0MjJkNGZkODNmM2YwZDllN2NlMDdkOTg4YmQxN2U2NjUzOTQ2NjhjNg==\",\"bXZSSnFNVE1mclkzS3RIMkE0cWRQZnEzUTZMNEt3OUNrNA==\"],\"Data\":\"RW50cnkgZGF0YTogYXNsO2RqZmFzbGRrZmphc2xkZmpsa3NvdWlld29wdXJ3Ig==\"}"
	fmt.Println(jsonstr)

	// Post the entry JSON to FactomClient web server	---------------------------------------
	data := url.Values{}
	data.Set("entry", jsonstr)
	data.Set("password", "opensesame")
	
	_, err = http.PostForm("http://localhost:8088/v1/submitentry", data)	
	if err != nil {
		t.Errorf("Error:%v", err)
	} else{
		fmt.Println("Entry successfully submitted to factomclient.")
	}		
	// JSON ws test done ----------------------------------------------------------------------------


	entry2 := new (Entry)
	reader := gocoding.ReadBytes([]byte(jsonstr))
	err = SafeUnmarshal(reader, entry2)
	
	fmt.Println("chainid:%v", base64.StdEncoding.EncodeToString(entry2.ChainID.Bytes))
	fmt.Println("ExtIDs0:%v", string(entry2.ExtIDs[0]))	
	fmt.Println("entrydata:%v", string(entry2.Data))	
			
	if err != nil {
		t.Errorf("Error:%v", err)
	}
} 

