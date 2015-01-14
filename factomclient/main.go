package main

import (
	"fmt"
	"github.com/FactomProject/dynrsrc"
	"github.com/FactomProject/gocoding"		
	"os"
	"io/ioutil"	
	"io"
	"log"
	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/database/ldb"		
	"github.com/FactomProject/FactomCode/factomapi"	
	"github.com/FactomProject/FactomCode/notaryapi"		
	"strings"
	"time"	
	"encoding/csv"
	"net/http"
	"net/url"	 
)  
 
var (
 	logLevel = "DEBUG"
	portNumber int = 8088 	
	applicationName = "factom/client"
	serverAddr = "localhost:8083"	
	ldbpath = "/tmp/factomclient/ldb9"	
	dataStorePath = "/tmp/factomclient/seed/csv"
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

func init() {

	loadConfigurations()
	
	initDB()
	
	factomapi.SetServerAddr(serverAddr)
	factomapi.SetDB(db)	
	
	err := dynrsrc.Start(watchError, readError)
	if err != nil { panic(err) }
	
	serve_init()
	initClientDataFileMap()	
	
	// Import data related to new factom blocks created on server
	ticker := time.NewTicker(time.Second * time.Duration(refreshInSeconds)) 
	go func() {
		for _ = range ticker.C {
			downloadAndImportDbRecords()
		}
	}()		
}

func main() {
	defer func() {
		dynrsrc.Stop()
		server.Close()
	}()
	
	server.Run(fmt.Sprint(":", portNumber))
	
}

func loadConfigurations(){
	cfg := struct {
		App struct{
			PortNumber	int		
			ApplicationName string
			ServerAddr string
			DataStorePath string
			RefreshInSeconds int
			LdbPath string
	    }
		Log struct{
	    	LogLevel string
		}
    }{}

	var  sf = "factomclient.conf"	
	wd, err := os.Getwd()
	if err != nil{
		log.Println(err)
	} else {
		sf =  wd+"/"+sf		
	}	

	err = gcfg.ReadFileInto(&cfg, sf)
	if err != nil{
		log.Println(err)
		log.Println("Client starting with default settings...")
	} else {
		log.Println("Client starting with settings from: " + sf)
		log.Println(cfg)
	
		//setting the variables by the valued form the config file
		logLevel = cfg.Log.LogLevel	
		applicationName = cfg.App.ApplicationName
		portNumber = cfg.App.PortNumber
		serverAddr = cfg.App.ServerAddr
		dataStorePath = cfg.App.DataStorePath
		refreshInSeconds = cfg.App.RefreshInSeconds
		log.Println(cfg.App.LdbPath)
		if cfg.App.LdbPath != "" {
			ldbpath = cfg.App.LdbPath
		}
	}
	
}

func initDB() {
	
	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	
	if err != nil{
		log.Println("err opening db: ", err)
	}
	
	if db == nil{
		log.Println("Creating new db ...")			
		db, err = ldb.OpenLevelDB(ldbpath, true)
		
		if err!=nil{
			panic(err)
		}		
	}
	
	log.Println("Database started from: " + ldbpath)

}

// to be replaced by DHT
func downloadAndImportDbRecords() {
	data := url.Values {}	
	data.Set("accept", "json")	
	data.Set("datatype", "filelist")
	data.Set("format", "binary")
	data.Set("password", "opensesame")	
	 
	server := fmt.Sprintf(`http://%s/v1`, serverAddr)
	resp, err := http.PostForm(server, data)
	
	if err != nil {
		fmt.Println("Error:", err)
		return
	} 	

	contents, _ := ioutil.ReadAll(resp.Body)
	if len(contents) < 5 {
		fmt.Println("The server file list is empty")
		return
	}
	
	serverMap := new (map[string]string)
	reader := gocoding.ReadBytes(contents)
	err = factomapi.SafeUnmarshal(reader, serverMap)	
	
	for key, value := range *serverMap {
		_, existing := clientDataFileMap[key]
		if !existing {
			data := url.Values {}	
			data.Set("accept", "json")	
			data.Set("datatype", "file")
			data.Set("filekey", key)
			data.Set("format", "binary")
			data.Set("password", "opensesame")	
			
			server := fmt.Sprintf(`http://%s/v1`, serverAddr)
			resp, err := http.PostForm(server, data)
			
			if fileNotExists( dataStorePath) {
				os.MkdirAll(dataStorePath, 0755)
			}	
			out, err := os.Create(dataStorePath + "/" + value)
			io.Copy(out, resp.Body)	
			out.Close()					
			
			// import records from the file into db
			file, err := os.Open(dataStorePath + "/" + value)
			if err != nil {panic(err)}
		    
		    reader := csv.NewReader(file) 	
		    //csv header: key, value
		    records, err := reader.ReadAll()	    
		    
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
	clientDataFileMap = make(map[string]string)	
	
	fiList, err := ioutil.ReadDir(dataStorePath)
	if err != nil {
		fmt.Println("Error in initServerDataFileMap:", err.Error())
		return err
	}

	for _, file := range fiList{
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			hash := notaryapi.Sha([]byte(file.Name()))
				
			clientDataFileMap[hash.String()] = file.Name()

		}
	}	
	return nil		
}

func fileNotExists(name string) (bool) {
  _, err := os.Stat(name)
  if os.IsNotExist(err) {
    return true
  }
  return err != nil
}