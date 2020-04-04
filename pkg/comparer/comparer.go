package comparer

import (
	"fmt"

	"github.com/Financial-Times/go-logger/v2"
)

// StartComparingResponses - starts a goroutine that listens for repeated requests and compares them
func StartComparingResponses(log *logger.UPPLogger) {
	go func() {

		for rrs := range repeaterChan {
			fmt.Printf("Got repeated response for tid:%+v\n", rrs.tid)

			logLine := log.WithTransactionID(rrs.tid)

			if len(rrs.responses) < 2 {
				logLine.Error(fmt.Errorf("Received less than two responses"))
				continue
			}

			b1 := rrs.responses[0].body
			b2 := rrs.responses[1].body
			if b1 != b2 {
				logLine.Error(fmt.Errorf("Responses do not match"))
				continue
			}

			// @todo - compare statusCodes
			// @todo - compare headers

			logLine.Debug("Responses were identical")
		}
	}()
}
