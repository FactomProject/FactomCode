package notaryapi

import (
	"testing"
	
	"fmt"
	"math"
	"crypto/sha256"
	"errors"
	
	"path/filepath"
	"sync"
	"io/ioutil"
	"encoding/hex"
	"time"
	
)


var blocks []*Block
var blockMutex = &sync.Mutex{}
var tickers [2]*time.Ticker

var data []string
var hashes []*Hash
var ebEntries []*EBEntry

// tested: in -> out (1st EBEntry with 25 ebEntries)
// tested: in2 -> out2 (fine, wrong with EBEntry.Marshall )
// tested: in2 -> out3 (fine), out3 -> out4
func TestWriteBlocks(t *testing.T) {
	init0() 
	createBlocks()

}

func init0() {
	
	// read store.0.block, use method from main.go
	initWithBinary()

	fmt.Println("Loaded", len(blocks), "blocks")

	for i := 0; i < len(blocks); i = i + 1 {
		if uint64(i) != blocks[i].BlockID {
			panic(errors.New("BlockID does not equal index"))
		}
	}
	UpdateNextBlockID(uint64(len(blocks)))
	
	initData()
}

	

func initData() {
	data = []string{"Beijing", "Shanghai", "Boston", "Atlanta", "Austin", "NYC", "Toronto",
		             "Dallas", "Houston", "Temple", "Seattle", "SF", "LA", "Hartford",
		             "Miami", "Baltimore", "New Heaven", "Portland", "Detroit", "Chicago"}

	hashes = make([]*Hash, len(data), len(data))
	
	ebEntries = make([]*EBEntry, len(data), len(data))

	for i := 0; i < len(data); i++ {
		
		//fmt.Println("i=", i, " data=", data[i])
		
		h := Sha(data[i])
		
		hashes[i] = h
		
		ebEntries[i] = NewEBEntry(h, UTF8DataType)
	}
}	

func createBlocks() {	
	for i := 0; i < len(ebEntries); i++ {
		
		fmt.Print("i=", i, " len(blocks)=", len(blocks))
		
		block := blocks[len(blocks)-1]
		
		fmt.Println(" new block id=", block.BlockID)
		
		block.AddEBEntry(ebEntries[i])
		
		if math.Mod(float64(i + 1), float64(5)) == 0 && i > 0 {
			// add a new block for new ebEntries to be added to
			fmt.Println("creating and adding a new block ...")
			blockMutex.Lock()
			newblock, err := CreateBlock(block, 10)
			
			if err != nil {fmt.Println("err...", err.Error())}
			if newblock == nil {fmt.Println("new block is nil")}
			
			blocks = append(blocks, newblock)
			blockMutex.Unlock()
		}
	}
	
	save()
}

func Sha(data string) (h *Hash) {
	sha := sha256.New()
	sha.Write([]byte(data))
	
	h = new(Hash)
	h.Bytes = sha.Sum(nil)
	return
	
}


func initWithBinary() {
	//matches, err := filepath.Glob(gobundle.DataFile("/home/jlu/temp/in/store.*.block"))
	matches, err := filepath.Glob("/home/jlu/temp/out3/store.*.block")
	if err != nil {
		panic(err)
	}

	blocks = make([]*Block, len(matches))
	fmt.Println("blocks.len=", len(blocks))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(Block)
		err = block.UnmarshalBinary(data)
		if err != nil {
			panic(err)
		}
		
		fmt.Println("index=", num, "block.id=", block.BlockID, " ebEntries.len=", len(block.ebEntries), "  prevHash=",  
			hex.EncodeToString(block.PreviousHash.Bytes), "  salt=", hex.EncodeToString(block.Salt.Bytes))

		blocks[num] = block
		num++
	}
}


func save() {
	bcp := make([]*Block, len(blocks))

	blockMutex.Lock()
	copy(bcp, blocks)
	blockMutex.Unlock()

	for i, block := range bcp {
		data, err := block.MarshalBinary()
		if err != nil {
			panic(err)
		}

		//err = ioutil.WriteFile(fmt.Sprintf(`app/rest/store.%d.block`, i), data, 0777)
		err = ioutil.WriteFile(fmt.Sprintf(`/home/jlu/temp/out4/store.%d.block`, i), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}
