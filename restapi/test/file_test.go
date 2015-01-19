package factomapi

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	//	"encoding/json"
	//	"encoding/base64"
	"io"
	"io/ioutil"
	"os"

	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/gocoding"
)

func TestGetFileList(t *testing.T) {
	data := url.Values{}
	data.Set("accept", "json")
	data.Set("datatype", "filelist")
	data.Set("format", "binary")
	data.Set("password", "opensesame")

	serverAddr := "localhost:8083"
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
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
	data.Set("filekey", "a48eac80b6c852261cd7beeb3fd50874a749b4c6c8a059d809c8552473f8a1ae")
	data.Set("format", "binary")
	data.Set("password", "opensesame")

	serverAddr := "localhost:8083"
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
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
