package main 

import (
	"flag"
	"net/http"
	"fmt"
)
var portNumber *int = flag.Bool("p", 8083, "Set the port to listen on")

// Default Request Handler
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>Hello %s!</h1>", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/", defaultHandler)
	http.ListenAndServe(":" + portNumber, nil)
}

