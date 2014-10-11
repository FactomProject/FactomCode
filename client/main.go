package main

import (
	//"flag"
	"fmt"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gobundle"
	"os"
	"log"
	"code.google.com/p/gcfg"
)

//var portNumber = flag.Int("p", 8087, "Set the port to listen on")
var (
 	logLevel = "DEBUG"
	portNumber int = 8087  	
	applicationName = "Factom/client"
)
func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init() {
	
	loadConfigurations()
	
	gobundle.Setup.Application.Name = applicationName
	gobundle.Init()
	
	err := dynrsrc.Start(watchError, readError)
	if err != nil { panic(err) }
	
	loadStore()
	loadSettings()
	templates_init()
	serve_init()
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
	}
	
}
