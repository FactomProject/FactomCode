package factomclient

import (
	"fmt"
	"os"
	"testing"
)

var _ = fmt.Sprint("")
var _ = os.DevNull

func TestLoadConfigurations(t *testing.T) {
	fmt.Println("LogLevel =", logLevel)
	fmt.Println("LogPath =", logPath)
}