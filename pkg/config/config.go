package config

import (
	"fmt"
	"net/url"

	cli "github.com/jawher/mow.cli"
)

// Config - struct to hold all config values
type Config struct {
	AppSystemCode       string
	LogLevel            string
	AppPort             int
	ProdServiceURL      *url.URL
	CandidateServiceURL *url.URL
	LogResponseBody     bool
}

// NewConfig - create new Config from env variables or command line parameters
func NewConfig(app *cli.Cli) (*Config, error) {

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

	appPort := app.Int(cli.IntOpt{
		Name:   "appPort",
		Value:  9999,
		Desc:   "Port to run the servicej on",
		EnvVar: "APP_PORT",
	})

	prodServiceURLStr := app.String(cli.StringOpt{
		Name:   "prodServiceURL",
		Desc:   "url for the production service: ex: http://localhost:9999",
		EnvVar: "PROD_SERVICE_URL",
	})

	candidateServiceURLStr := app.String(cli.StringOpt{
		Name:   "candidateServiceURL",
		Desc:   "url for the candidate service: ex: http://localhost:9999",
		EnvVar: "CANDIDATE_SERVICE_URL",
	})

	logResponseBody := app.Bool(cli.BoolOpt{
		Name:   "logResponseBody",
		Value:  false,
		Desc:   "wheter or not to log the body of the responses",
		EnvVar: "LOG_RESPONSE_BODY",
	})

	prodServiceURL, err := url.Parse(*prodServiceURLStr)
	if err != nil {
		return nil, fmt.Errorf("incorrect value for prodServiceURL: %w", err)
	}
	candidateServiceURL, err := url.Parse(*candidateServiceURLStr)
	if err != nil {
		return nil, fmt.Errorf("incorrect value for candidateServiceURL: %w", err)
	}

	return &Config{
		AppSystemCode:       *appSystemCode,
		LogLevel:            *logLevel,
		AppPort:             *appPort,
		ProdServiceURL:      prodServiceURL,
		CandidateServiceURL: candidateServiceURL,
		LogResponseBody:     *logResponseBody,
	}, nil
}
