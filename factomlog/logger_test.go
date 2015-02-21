package factomlog

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer

	logger := New(&buf, "debug", "testing")

	logger.Info("Hello Log!")
	logger.Debug("Hello Log!")

	fmt.Print(&buf)
}