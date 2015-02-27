package factomclient

import (
	"fmt"
	"os"
	"testing"
	
	"github.com/FactomProject/FactomCode/util"
)

var _ = fmt.Sprint("")
var _ = os.DevNull

func TestLoadConfigurations(t *testing.T) {
	cfg := util.ReadConfig()
	fmt.Printf("%v\n", cfg)
	fmt.Println("LogLevel =", cfg.Log.LogLevel)
	fmt.Println("LogPath =", cfg.Log.LogPath)
}