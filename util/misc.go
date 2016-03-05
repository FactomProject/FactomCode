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

// Calculate the entry credits needed for the entry
func EntryCost(b []byte) (uint8, error) {

	// caulculaate the length exluding the header size 35 for Milestone 1
	l := len(b) - 35

	if l > 10240 {
		return 10, fmt.Errorf("Entry cannot be larger than 10KB")
	}

	// n is the capacity of the entry payment in KB
	r := l % 1024
	n := uint8(l / 1024)

	if r > 0 {
		n += 1
	}

	if n < 1 {
		n = 1
	}

	return n, nil
}
