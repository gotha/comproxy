package comparer

import (
	"fmt"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/davecgh/go-spew/spew"
)

// StartComparingResponses - starts a goroutine that listens for repeated requests and compares them
func StartComparingResponses(log *logger.UPPLogger) {
	go func() {

		for rrs := range repeaterChan {
			fmt.Printf("Got repeated response for tid:%+v\n", rrs.tid)
			spew.Dump(rrs)
		}
	}()
}
