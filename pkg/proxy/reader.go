package proxy

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type key string

const (
	ReqBodyKey key = "reqBody"
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
	// @todo - this doesnt seem to work - fixit
	rw.StatusCode = statusCode
	rw.orig.WriteHeader(statusCode)
}

// RequestBodyReader - reads body and adds it into the request context
func RequestBodyReader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

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

		reqWithBody := req.WithContext(context.WithValue(req.Context(), ReqBodyKey, body))

		next.ServeHTTP(rw, reqWithBody)
	})
}
