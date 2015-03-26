package wsapi

import (
	"fmt"
	"os"
	"testing"
	"time"
	
	"github.com/FactomProject/FactomCode/database"
	"github.com/FactomProject/FactomCode/factomwire"
	"github.com/FactomProject/FactomCode/util"
)

var _ = fmt.Sprint("")
var _ = os.DevNull

func TestStart(t *testing.T) {
	var db database.Db
	outMsgQ := make(chan factomwire.Message)
	
	cfg := util.ReadConfig().Wsapi
	fmt.Printf("%v\n", cfg)
	fmt.Println("wsapi.Start")
	Start(db, outMsgQ)
	fmt.Println("in parallel!")
	time.Sleep(30 * time.Second)
	fmt.Println("wsapi.Stop")
	Stop()	
}