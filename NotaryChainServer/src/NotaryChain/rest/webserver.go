package main 

import (
	"flag"
	"net/http"
	"fmt"
	ncdata "NotaryChain/data"
	"strconv"
)

var portNumber *int = flag.Int("p", 8083, "Set the port to listen on")

// Default Request Handler
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>Hello %s!</h1>", r.URL.Path[1:])
}

var blocks [10]ncdata.Block

func main() {
	e := [10]ncdata.Entry{}
	
	http.HandleFunc("/", defaultHandler)
	http.ListenAndServe(":" + strconv.Itoa(*portNumber), nil)
}