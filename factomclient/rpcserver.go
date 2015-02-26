package factomclient

import (
	"fmt"
	"github.com/FactomProject/dynrsrc"
	"github.com/FactomProject/gocoding"
	"io"
	"io/ioutil"
	"os"
	//	"log"
	//	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"
	//	"github.com/FactomProject/FactomCode/database/ldb"
	"encoding/csv"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/util"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	portNumber       int = 8088
	applicationName      = "factom/client"
	serverAddr           = "localhost:8083"
	//ldbpath              = "/tmp/factomclient/ldb9"
	dataStorePath        = "/tmp/store/seed/csv"
	refreshInSeconds int = 60

	db database.Db // database

	//Map to store imported csv files
	clientDataFileMap map[string]string
)

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init_rpcserver() {
	log := rpcLog
	log.Debug("dynrsrc.Start")
	err := dynrsrc.Start(watchError, readError)
	if err != nil {
		log.Alert(err)
	}

	log.Info("Starting server")
	serve_init()
	//initClientDataFileMap()

	// Import data related to new factom blocks created on server
	ticker := time.NewTicker(time.Second * time.Duration(refreshInSeconds))
	go func() {
		for _ = range ticker.C {
			//downloadAndImportDbRecords()
		}
	}()
}

func Start_Rpcserver(ldb database.Db, outMsgQ chan<- factomwire.Message) {
	db = ldb
	factomapi.SetDB(db)
	factomapi.SetOutMsgQueue(outMsgQ)

	rpcLog.Info("Starting rpcserver")
	init_rpcserver()

	defer func() {
		dynrsrc.Stop()
		server.Close()
	}()

	server.Run(fmt.Sprint(":", portNumber))

}

func LoadConfigurations(cfg *util.FactomdConfig) {
	applicationName = cfg.Rpc.ApplicationName
	portNumber = cfg.Rpc.PortNumber
	dataStorePath = cfg.App.DataStorePath
	refreshInSeconds = cfg.Rpc.RefreshInSeconds
}

// to be replaced by DHT
func downloadAndImportDbRecords() {
	log := rpcLog

	data := url.Values{}
	data.Set("accept", "json")
	data.Set("datatype", "filelist")
	data.Set("format", "binary")
	data.Set("password", "opensesame")

	server := fmt.Sprintf(`http://%s/v1/getfilelist`, serverAddr)
	resp, err := http.PostForm(server, data)

	if err != nil {
		log.Error(err)
		return
	}

	contents, _ := ioutil.ReadAll(resp.Body)
	if len(contents) < 5 {
		log.Notice("The server file list is empty")
		return
	}

	serverMap := new(map[string]string)
	reader := gocoding.ReadBytes(contents)
	err = factomapi.SafeUnmarshal(reader, serverMap)

	for key, value := range *serverMap {
		_, existing := clientDataFileMap[key]
		if !existing {
			data := url.Values{}
			data.Set("accept", "json")
			data.Set("datatype", "file")
			data.Set("filekey", key)
			data.Set("format", "binary")
			data.Set("password", "opensesame")

			server := fmt.Sprintf(`http://%s/v1/getfile`, serverAddr)
			resp, err := http.PostForm(server, data)
			if err != nil {
				log.Error(err)
			}

			if fileNotExists(dataStorePath) {
				log.Infof("%s does not exist. creating it now.", dataStorePath)
				os.MkdirAll(dataStorePath, 0755)
			}
			out, err := os.Create(dataStorePath + "/" + value)
			if err != nil {
				log.Error(err)
			}
			io.Copy(out, resp.Body)
			out.Close()

			// import records from the file into db
			file, err := os.Open(dataStorePath + "/" + value)
			if err != nil {
				log.Alert(err)
			}

			reader := csv.NewReader(file)
			//csv header: key, value
			records, err := reader.ReadAll()
			if err != nil {
				log.Error(err)
			}

			var ldbMap = make(map[string]string)
			for _, record := range records {
				ldbMap[record[0]] = record[1]
			}
			db.InsertAllDBRecords(ldbMap)
			file.Close()

			// add the file to the imported list
			clientDataFileMap[key] = value
		}
	}
}

// Initialize the imported file list
func initClientDataFileMap() error {
	log := rpcLog
	clientDataFileMap = make(map[string]string)

	fiList, err := ioutil.ReadDir(dataStorePath)
	if err != nil {
		log.Error("initServerDataFileMap:", err)
		return err
	}

	for _, file := range fiList {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			hash := notaryapi.Sha([]byte(file.Name()))

			clientDataFileMap[hash.String()] = file.Name()

		}
	}
	return nil
}

func fileNotExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return true
	}
	return err != nil
}
