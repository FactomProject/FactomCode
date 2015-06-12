package factomapi_test

import (
	"testing"
	"fmt"
	
	"github.com/FactomProject/FactomCode/factomapi"
)

func TestDBlockHead(t *testing.T) {
	d, err := factomapi.DBlockHead()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%#v\n", d)
}
