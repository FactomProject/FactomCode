package util

import (
	"fmt"
	"runtime"
	"time"
)

// a simple file/line trace function, with optional comment(s)
func Trace(params ...string) {
	fmt.Printf("##")

	if 0 < len(params) {
		for i := range params {
			fmt.Printf(" %s", params[i])
		}
		fmt.Printf(" #### ")
	} else {
		fmt.Printf(" ")
	}

	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])

	tutc := time.Now().UTC()
	timestamp := tutc.Format("2006-01-02.15:04:05")

	fmt.Printf("TRACE: %s line %d %s file: %s\n", timestamp, line, f.Name(), file)
}
