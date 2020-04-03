package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	cli "github.com/jawher/mow.cli"
)

func main() {

	app := cli.App("comproxy", "Description")

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "comproxy",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Logging level (DEBUG, INFO, WARN, ERROR)",
		EnvVar: "LOG_LEVEL",
	})

	appPort := app.String(cli.StringOpt{
		Name:   "appPort",
		Value:  "9999",
		Desc:   "Port to run the servicej on",
		EnvVar: "APP_PORT",
	})

	prodServiceURL := app.String(cli.StringOpt{
		Name:   "prodServiceURL",
		Desc:   "url for the production service: ex: http://localhost:9999",
		EnvVar: "PROD_SERVICE_URL",
	})

	candidateServiceURL := app.String(cli.StringOpt{
		Name:   "candidateServiceURL",
		Desc:   "url for the candidate service: ex: http://localhost:9999",
		EnvVar: "CANDIDATE_SERVICE_URL",
	})

	log := logger.NewUPPLogger(*appSystemCode, *logLevel)
	log.Infof("[Startup] %s is starting ", *appSystemCode)

	s := Services{
		Primary: Service{
			url: *prodServiceURL,
		},
		Candidate: Service{
			url: *candidateServiceURL,
		},
	}
	h := NewHandler(s, log)
	h.StartRepeater()
	h.StartComparer()
	h.StartCleaner()

	proxy, err := h.GetProxy()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", *appPort),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	http.HandleFunc("/", proxy)
	srv.SetKeepAlivesEnabled(false)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}
