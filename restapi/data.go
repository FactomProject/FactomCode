package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/FactomProject/FactomCode/notaryapi"
)

func doEntry(cid *notaryapi.Hash, data []byte) {
	fmt.Println("doEntry: ", string(data))

	entry := new(notaryapi.Entry)
	var hash = notaryapi.Hash{Bytes: cid.Bytes}
	entry.ChainID = hash
	entry.Data = data

	_, err := processRevealEntry(entry) //Need to update ??
	if err != nil {
		fmt.Println(err.Error())
	}
}

func doEntries() {
	fmt.Println("** chainmap.len=", len(chainIDMap))

	chash := make([]*notaryapi.Hash, 0, len(chainIDMap))
	for _, v := range chainIDMap {
		chash = append(chash, v.ChainID)
		if v.ChainID == nil {
			fmt.Println("chainid is nil")
		}
	}

	for i := 0; i < 10; i++ { //10 round of 2 minutes cycle to generate FB
		for j := 0; j < 5; j++ { //5 entries generated per minute
			doEntry(chash[0], []byte("Apple."+strconv.Itoa(i)+"."+strconv.Itoa(j)))
			doEntry(chash[1], []byte("Banana."+strconv.Itoa(i)+"."+strconv.Itoa(j)))
			time.Sleep(12 * time.Second)
		}
	}

	time.Sleep(15 * time.Minute)
}
