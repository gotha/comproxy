package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/dchest/uniuri"
)

// RequestLoggger - using UPPLogger logs every request
func RequestLoggger(next http.Handler, log *logger.UPPLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		var body string
		if req.Body != nil {

			var buf1, buf2 bytes.Buffer
			w := io.MultiWriter(&buf1, &buf2)

			if _, err := io.Copy(w, req.Body); err != nil {
				log.Fatal(err)
			}

			reader := bytes.NewReader(buf1.Bytes())
			req.Body = ioutil.NopCloser(reader)
			body = buf2.String()
		}

		tid := req.Header.Get("X-Request-Id")
		if tid == "" {
			tid = fmt.Sprintf("tid_%s", uniuri.NewLen(16))
			req.Header.Set("X-Request-Id", tid)
		}

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

		tid := req.Header.Get("X-Request-Id")
		var body string
		// @todo - make this configuration option
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
