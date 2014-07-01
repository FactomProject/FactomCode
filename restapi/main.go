package main

/*
#cgo darwin CFLAGS: -I/usr/local/Cellar/readline/6.2.4/include/
#cgo darwin LDFLAGS: -L/usr/local/Cellar/readline/6.2.4/lib/
*/
import (
	"fmt"
	"github.com/fiorix/go-readline"
	"github.com/firelizzard18/gobundle"
)

func init() {
	gobundle.Setup.Application.Name = "NotaryChains/restapi"
	gobundle.Init()
	
	loadBlocks()
	templates_init()
	saveBlocks_init()
	btc_init()
	serve_init()
	cmd_init()
	
	fmt.Println("Loaded", len(blocks), "blocks")
}

func fini() {
	templates_fini()
	saveBlocks_fini()
	btc_fini()
	serve_fini()
	
	fmt.Println("Shutting down")
}

func completer(input, line string, start, end int) []string {
	nn, rest := commands.GetPartial(input)
	
	if nn == nil || len(rest) > 0 {
		return []string{}
	}
	
	arr := []string{}
	for _, key := range nn.Keys() {
		arr = append(arr, input + key)
	}
	return arr
}

func main() {
	go serve_main()
	
	prompt := "> "

//	readline.SetCompletionFunction(completer)

	// This is generally what people expect in a modern Readline-based app
	readline.ParseAndBind("TAB: menu-complete")

	// Loop until Readline returns nil (signalling EOF)
L:
	for {
		result := readline.Readline(&prompt)
		switch {
		case result == nil:
			fmt.Println()
			break L // exit loop
			
		case *result == "exit":
			break L // exit loop
			
		case *result != "": // Ignore blank lines
			fmt.Println(*result)
//			readline.AddHistory(*result) // Allow user to recall this line
		}
	}
}