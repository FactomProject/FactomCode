package main

import (
	"errors"
	"fmt"
	"github.com/firelizzard18/dynrsrc"
	"github.com/firelizzard18/gobundle"
	"github.com/NotaryChains/NotaryChainCode/notaryapi"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"
)

var blocks []*notaryapi.Block
var blockMutex = &sync.Mutex{}

var saveTicker = time.NewTicker(time.Hour)

func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func templates_init() {
	dynrsrc.Start(watchError, readError)
	notaryapi.StartDynamic(gobundle.DataFile("html.gwp"), readError)
}

func templates_fini() {
	dynrsrc.Stop()
}

func loadBlocks() {
	matches, err := filepath.Glob(gobundle.DataFile("store.*.block"))
	if err != nil {
		panic(err)
	}

	blocks = make([]*notaryapi.Block, len(matches))

	num := 0
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			panic(err)
		}

		block := new(notaryapi.Block)
		err = block.UnmarshalBinary(data)
		if err != nil {
			panic(err)
		}

		blocks[num] = block
		num++
	}

	for i := 0; i < len(blocks); i = i + 1 {
		if uint64(i) != blocks[i].BlockID {
			panic(errors.New("BlockID does not equal index"))
		}
	}
	notaryapi.UpdateNextBlockID(uint64(len(blocks)))
}

func saveBlocks_init() {
	go func() {
		for _ = range saveTicker.C {
			saveBlocks()
		}
	}()
}

func saveBlocks_fini() {
	saveTicker.Stop()
	saveBlocks()
}

func saveBlocks() {
	bcp := make([]*notaryapi.Block, len(blocks))

	blockMutex.Lock()
	copy(bcp, blocks)
	blockMutex.Unlock()

	for i, block := range bcp {
		data, err := block.MarshalBinary()
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(gobundle.DataFile(fmt.Sprintf(`store.%d.block`, i)), data, 0777)
		if err != nil {
			panic(err)
		}
	}
}