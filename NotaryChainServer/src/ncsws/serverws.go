package main 

import (
	stdlog "log"
	"net/http"
	"os"
	"html/template"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("NotaryChain.server.ws")
var tmpl *template.Template
var err error

// Default Request Handler
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	err = tmpl.ExecuteTemplate(w, "html.gwsp", r);
	if err != nil { log.Panic(err) }
}

func main() {
	logging.SetFormatter(logging.MustStringFormatter("%{level:.1s} 0x%{id:x} %{message}"))
	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetLevel(logging.INFO, "NotaryChain.server.ws")
	
	log.Info("Initializing")
	
	log.Info("Parsing Go WebServer Pages")
	tmpl, err = template.ParseGlob("*.gwsp")
	if err != nil { log.Panic(err) }
	
	log.Info("Listening")
	http.HandleFunc("/", defaultHandler)
	http.ListenAndServe(":8080", nil)
}

