package main

import (
	//"flag"
	"fmt"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gobundle"
	"os"
	"io/ioutil"	
	"log"
	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/database/ldb"		
	"strings"
	"time"	
	"encoding/csv"
)

//var portNumber = flag.Int("p", 8087, "Set the port to listen on")
var (
 	logLevel = "DEBUG"
	portNumber int = 8087  	
	applicationName = "factom/client"
	serverAddr = "localhost:8083"	
	ldbpath = "/tmp/client/ldb9"	
	dataStorePath = "/tmp/store/seed/csv"
	refreshInSeconds int = 60
	
	db database.Db // database
	
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
		
	gobundle.Setup.Application.Name = applicationName
	gobundle.Init()
	
	err := dynrsrc.Start(watchError, readError)
	if err != nil { panic(err) }
	
	loadStore()
	loadSettings()
	templates_init()
	serve_init()
	
	// Import data related to new factom blocks created on server
	ticker := time.NewTicker(time.Second * time.Duration(refreshInSeconds)) 
	go func() {
		for _ = range ticker.C {
			ImportDbRecordsFromFile()
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
	    }
		Log struct{
	    	LogLevel string
		}
    }{}
	
	wd, err := os.Getwd()
	if err != nil{
		log.Println(err)
	}	
	err = gcfg.ReadFileInto(&cfg, wd+"/client.conf")
	if err != nil{
		log.Println(err)
		log.Println("Client starting with default settings...")
	} else {
	
		//setting the variables by the valued form the config file
		logLevel = cfg.Log.LogLevel	
		applicationName = cfg.App.ApplicationName
		portNumber = cfg.App.PortNumber
		serverAddr = cfg.App.ServerAddr
		dataStorePath = cfg.App.DataStorePath
		refreshInSeconds = cfg.App.RefreshInSeconds
	}
	
}

func initDB() {
	
	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	
	if err != nil{
		log.Println("err opening db: %v", err)
	}
	
	if db == nil{
		log.Println("Creating new db ...")			
		db, err = ldb.OpenLevelDB(ldbpath, true)
		
		if err!=nil{
			panic(err)
		} else{
			log.Println("Database started from: " + ldbpath)
		}		
	}
	

}

func ImportDbRecordsFromFile() {

 	fileList, err := getCSVFiles()
 	
 	if err != nil{
 		log.Println(err)
 		return
 	}
 	 	
 	for _, filePath := range fileList{
 		
		file, err := os.Open(filePath)
		if err != nil {panic(err)}
	    defer file.Close()
	    
	    reader := csv.NewReader(file) 	
	    //csv header: key, value
	    records, err := reader.ReadAll()	    
	    
	    var ldbMap = make(map[string]string)	
		for _, record := range records {
			ldbMap[record[0]] = record[1]
		}	    	
	 	db.InsertAllDBRecords(ldbMap)   
			
		// rename the processed file
		os.Rename(filePath, filePath + "." + time.Now().Format(time.RFC3339))	
	}
			
}

// get csv files from csv directory
func getCSVFiles() (fileList []string, err error) {

	fiList, err := ioutil.ReadDir(dataStorePath)
	if err != nil {
		return nil, err
	}
	fileList = make ([]string, 0, 10)

	for _, file := range fiList{
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			fileList = append(fileList, "/tmp/store/seed/csv/" + file.Name())
		}
	}	
	return fileList, nil
}

