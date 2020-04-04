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

/*
func (h *Handler) StartRepeater() {

	URL, err := url.Parse(h.Services.Candidate.url)
	if err != nil {
		h.log.Fatalf("Could not initialise repeater")
		return
	}

	go func() {
		for rec := range newRecords {

			h.log.Debug("Received new record; Repeating request to candidate service")

			var req http.Request
			copier.Copy(&req, &rec.req)

			req.Host = URL.Host
			req.URL.Host = URL.Host
			req.URL.Scheme = URL.Scheme
			req.RequestURI = ""

			httpClient := NewHTTPClient()
			resp, err := httpClient.Do(&req)
			if err != nil {
				h.log.Errorf("Could not send repeater request", err)
				return
			}

			br, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				h.log.Errorf("Could not read body of repeater response", err)
				return
			}

			// write response to chan
			streader := bytes.NewReader(br)
			respBody, err := ioutil.ReadAll(streader)
			if err != nil {
				h.log.Errorf("Could not read body of repeater response", err)
			}
			rec.addResponse(Response{
				body:    respBody,
				headers: resp.Header,
			})

			resp.Body.Close()

		}
	}()
}

/*
func (h *Handler) StartComparer() {
	go func() {
		for rec := range newResponses {

			h.log.
				WithTransactionID(rec.tid).
				Info("Received response")

			if len(rec.responses) < 2 {
				continue
			}

			b1 := string(rec.responses[0].body)
			b2 := string(rec.responses[1].body)
			if b1 != b2 {
				h.log.
					WithTransactionID(rec.tid).
					Error("Different results were returned")
				continue
			}

			h.log.
				WithTransactionID(rec.tid).
				Debug("results were idential")

			h.store.removeRecord(rec.stamp)
			h.log.
				WithTransactionID(rec.tid).
				Debug("Removing record since responses were compared")

		}
	}()
}
*/

/**
func (h *Handler) StartCleaner() {
	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				t := time.Now().UnixNano()
				since := t - 5*1000000000
				records := h.store.getRecordsOlderThan(since)

				h.log.Debugf("Cleaner found %d records older than 5 sec", len(records))
				for _, rec := range records {
					numResponses := len(rec.responses)
					if numResponses == 2 {
						h.store.removeRecord(rec.stamp)
						h.log.Debug("Removed old record with 2 responses")
						continue
					}
					if numResponses < 2 {
						h.log.
							WithTransactionID(rec.tid).
							Error("Less than 2 responses were recorded; one the services might be faulty")
						h.store.removeRecord(rec.stamp)
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
*/
