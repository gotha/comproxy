package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/gotha/comproxy/pkg/comparer"
	"github.com/gotha/comproxy/pkg/config"
	"github.com/gotha/comproxy/pkg/proxy"

	cli "github.com/jawher/mow.cli"
)

func main() {

	app := cli.App("comproxy", "Description")

	c, err := config.NewConfig(app)
	if err != nil {
		fmt.Printf("Could not start app because of cofiguration error: %w", err)
		os.Exit(1)
	}

	log := logger.NewUPPLogger(c.AppSystemCode, c.LogLevel)
	log.Infof("[Startup] %s is starting ", c.AppSystemCode)

	var handler http.Handler
	handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("%+v\n", "Journey ends here")
	})
	handler = comparer.Bridge(handler)
	handler = proxy.ResponseLogger(handler, log, c.LogResponseBody)
	handler = proxy.NewHandler(handler, c.ProdServiceURL)
	handler = proxy.ResponseReader(handler)
	handler = proxy.RequestLoggger(handler, log)
	handler = proxy.RequestBodyReader(handler)
	handler = proxy.TransactionHandler(handler)

	http.Handle("/", handler)

	comparer.StartRepeater(c.CandidateServiceURL, log, c.LogResponseBody)
	comparer.StartComparingResponses(log)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", c.AppPort),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	srv.SetKeepAlivesEnabled(false)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}
