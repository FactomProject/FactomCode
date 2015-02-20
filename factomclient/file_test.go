package factomclient

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	//	"encoding/json"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/gocoding"
	//	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
)

func TestGetFileList(t *testing.T) {
	data := url.Values{}
	data.Set("accept", "json")
	data.Set("datatype", "filelist")
	data.Set("format", "binary")
	data.Set("password", "opensesame")

	serverAddr := "localhost:8088"
	server := fmt.Sprintf(`http://%s/v1/getfilelist`, serverAddr)
	resp, err := http.PostForm(server, data)

	if err != nil {
		t.Errorf("Error:%v", err)
	} else {
		fmt.Println("Request dblocksbyrange successfully submitted to factomclient.")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("after contents: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Http Resp Body:%s\n", string(contents))
	fmt.Println("status code:%v", resp.StatusCode)

	if len(contents) < 5 {
		fmt.Println("The file list is empty")
		return
	}

	serverMap := new(map[string]string)
	reader := gocoding.ReadBytes(contents)
	err = factomapi.SafeUnmarshal(reader, serverMap)

	for key, value := range *serverMap {
		fmt.Println("Key:", key, "Value:", value)
	}

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}

func TestGetFile(t *testing.T) {
	data := url.Values{}
	data.Set("accept", "json")
	data.Set("datatype", "file")
	data.Set("filekey", "cb51f551cb6410618be55482b4f2e9750ae915d99592bc23d4feeb453e88d5fa")
	data.Set("format", "binary")
	data.Set("password", "opensesame")

	serverAddr := "localhost:8088"
	server := fmt.Sprintf(`http://%s/v1/getfile`, serverAddr)
	resp, err := http.PostForm(server, data)

	if err != nil {
		t.Errorf("Error:%v", err)
	} else {
		fmt.Println("Request dblocksbyrange successfully submitted to factomclient.")
	}
	out, err := os.Create("/tmp/output.txt")
	defer out.Close()

	n, err := io.Copy(out, resp.Body)

	fmt.Println("status code:%v", resp.StatusCode)
	fmt.Println("File create with bytes:", n)

	if err != nil {
		t.Errorf("Error:%v", err)
	}

}
