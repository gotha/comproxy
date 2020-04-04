package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	_ "github.com/davecgh/go-spew/spew"
	httpclient "github.com/gotha/comproxy/pkg/http"
)

func NewHandler(next http.Handler, URL *url.URL) http.Handler {

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		// modify request to go to new target
		req.Host = URL.Host
		req.URL.Host = URL.Host
		req.URL.Scheme = URL.Scheme
		req.RequestURI = ""

		httpClient := httpclient.NewHTTPClient()
		resp, err := httpClient.Do(req)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}
		defer resp.Body.Close()

		// write response headers
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		rw.Header().Set("X-Forwarded-For", host)
		for key, values := range resp.Header {
			for _, value := range values {
				rw.Header().Set(key, value)
			}
		}

		// send trailer header
		trailerKeys := []string{}
		for k := range resp.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		if len(trailerKeys) > 0 {
			rw.Header().Set("Trailer", strings.Join(trailerKeys, ","))
		}

		// write body and copy content so it can be added to store
		rw.WriteHeader(resp.StatusCode)
		_, err = io.Copy(rw, resp.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}

		// copy trailer
		for k, values := range resp.Trailer {
			for _, v := range values {
				rw.Header().Set(k, v)
			}
		}

		next.ServeHTTP(rw, req)
	})
}
