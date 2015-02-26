package factomlog

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer

	name := "Michael"

	logger := New(&buf, "info", "testing")

	logger.Infof("Hello %s!", name)
	logger.Info("Hello ", name)
	logger.Debug("Hello Log!")

	fmt.Print(&buf)
}
