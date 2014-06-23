package main

import (
	"flag"
	"fmt"
	
	"github.com/firelizzard18/dynrsrc"
)

var portNumber = flag.Int("p", 8087, "Set the port to listen on")
var appDir = flag.String("C", "app/client", "Set the app directory")

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init() {
	err := dynrsrc.Start(watchError, readError)
	if err != nil { panic(err) }
	
	flag.Parse()
	
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