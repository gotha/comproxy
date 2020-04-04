package proxy

import (
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
)

// RequestLoggger - using UPPLogger logs every request
func RequestLoggger(next http.Handler, log *logger.UPPLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		tid := req.Header.Get(TransactionIDGHeader)
		body := req.Context().Value(ReqBodyKey)

		log.
			WithField("method", req.Method).
			WithField("url", req.URL.String()).
			WithField("body", body).
			WithTransactionID(tid).
			Info("Received request")

		next.ServeHTTP(w, req)
	})
}

// ResponseLogger - logs response for specific request
func ResponseLogger(next http.Handler, log *logger.UPPLogger, logBody bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		tid := req.Header.Get(TransactionIDGHeader)
		var body string
		if logBody == true {
			body = w.(ResponseWriterWithValue).Val.String()
		}

		log.
			WithField("method", req.Method).
			WithField("url", req.URL.String()).
			WithField("body", body).
			WithTransactionID(tid).
			Info("Returned response")

		next.ServeHTTP(w, req)
	})
}
