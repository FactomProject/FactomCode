package main 

import (
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"github.com/firelizzard18/gocoding/json"
	"io/ioutil"
	"os"
)

func main() {
	blocks := []*notaryapi.Block{new(notaryapi.Block),new(notaryapi.Block)}
	
	data, err := ioutil.ReadFile(`app/rest/store.0.block`)
	if err != nil { panic(err) }
	
	err = blocks[0].UnmarshalBinary(data)
	if err != nil { panic(err) }
	
	data, err = ioutil.ReadFile(`app/rest/store.1.block`)
	if err != nil { panic(err) }
	
	err = blocks[1].UnmarshalBinary(data)
	if err != nil { panic(err) }
	
	marshaller := json.NewIndentedMarshaller(os.Stdout, "", "\t")
	
	err = marshaller.Marshal(blocks)
	if err != nil { panic(err) }
}
