package proxy

import (
	"bytes"
	"io"
	"net/http"
)

// ResponseReader - replaces ResponseWriter with ResponseWriterWithValue and when written stores the value of the response to val property
func ResponseReader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var logResp bytes.Buffer
		mw := io.MultiWriter(rw, &logResp)

		w := ResponseWriterWithValue{
			source: mw,
			orig:   rw,
			Val:    &logResp,
		}
		next.ServeHTTP(w, req)
	})
}

type ResponseWriterWithValue struct {
	source     io.Writer
	orig       http.ResponseWriter
	Val        *bytes.Buffer
	StatusCode int
}

func (rw ResponseWriterWithValue) Write(p []byte) (int, error) {
	return rw.source.Write(p)
}

func (rw ResponseWriterWithValue) Header() http.Header {
	return rw.orig.Header()
}

func (rw ResponseWriterWithValue) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
	rw.orig.WriteHeader(statusCode)
}
