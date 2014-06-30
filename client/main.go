package main

import (
	"flag"
	"fmt"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gobundle"
)

var portNumber = flag.Int("p", 8087, "Set the port to listen on")

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init() {
	gobundle.Setup.Application.Name = "NotaryChains/client"
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
	
	server.Run(fmt.Sprint(":", *portNumber))
}
