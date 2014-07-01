package main

import (
	"fmt"
)

var commands = &Node{}

func cmd_init() {
	commands.Put("save", cmd_save)
}

func cmd_save() error {
	fmt.Println("Saving blocks")
	saveBlocks()
	fmt.Println("Saved blocks")
	return nil
}