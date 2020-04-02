package main

import (
	"net/http"

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
	http.ListenAndServe(":8080", proxy)

}

//func waitForSignal() {
//	ch := make(chan os.Signal, 1)
//	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
//	<-ch
//}
