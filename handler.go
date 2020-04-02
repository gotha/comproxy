package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger/v2"
)

type Service struct {
	url string
}

type Services struct {
	Primary   Service
	Candidate Service
}

type Handler struct {
	log       *logger.UPPLogger
	Services  Services
	Requests  chan StampedRequest
	Responses chan StampedResponse
	store     Store
}

func NewHandler(s Services, log *logger.UPPLogger) Handler {
	return Handler{
		Services:  s,
		log:       log,
		Requests:  make(chan StampedRequest, 1000),
		Responses: make(chan StampedResponse, 1000),
		store:     NewStore(),
	}
}

func (h *Handler) GetProxy() (http.HandlerFunc, error) {

	URL, err := url.Parse(h.Services.Primary.url)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		streq := NewStampedRequest(*req)
		// @todo - log body if the request is POST
		h.log.
			WithFields(streq.GetProperties()).
			WithTransactionID(streq.tid).
			Info("Received request")

		h.Requests <- streq
		h.store.addRequest(streq)

		r, err := copyRequest(*req)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}

		r.Host = URL.Host
		r.URL.Host = URL.Host
		r.URL.Scheme = URL.Scheme
		r.RequestURI = ""

		httpClient := NewHTTPClient()
		resp, err := httpClient.Do(&r)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}

		br, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			return
		}

		// write headers
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		rw.Header().Set("X-Forwarded-For", host)

		for key, values := range resp.Header {
			for _, value := range values {
				rw.Header().Set(key, value)
			}
		}

		// we do this do handle streams
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-time.Tick(10 * time.Millisecond):
					rw.(http.Flusher).Flush()
				case <-done:
					return
				}
			}
		}()

		// send trailer header
		trailerKeys := []string{}
		for k := range resp.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		if len(trailerKeys) > 0 {
			rw.Header().Set("Trailer", strings.Join(trailerKeys, ","))
		}

		// write body
		rw.WriteHeader(resp.StatusCode)
		reader := bytes.NewReader(br)
		io.Copy(rw, reader)

		// copy trailer
		for k, values := range resp.Trailer {
			for _, v := range values {
				rw.Header().Set(k, v)
			}
		}

		// writte response to chan
		streader := bytes.NewReader(br)
		respBody, err := ioutil.ReadAll(streader)
		if err != nil {
			log.Fatal(err)
		}
		stresp := NewStampedResponse(respBody, resp.Header, streq)
		h.Responses <- stresp
		h.store.addResponse(stresp)

		close(done)

	}), nil
}

func (h *Handler) StartRepeater() {

	URL, err := url.Parse(h.Services.Candidate.url)
	if err != nil {
		h.log.Fatalf("Could not initialise repeater")
		return
	}

	go func() {
		for streq := range h.Requests {

			r, err := copyRequest(streq.req)
			if err != nil {
				h.log.Errorf("Could not copy request", err)
				return
			}

			r.Host = URL.Host
			r.URL.Host = URL.Host
			r.URL.Scheme = URL.Scheme
			r.RequestURI = ""

			httpClient := NewHTTPClient()
			resp, err := httpClient.Do(&r)
			if err != nil {
				h.log.Errorf("Could not send repeater request", err)
				return
			}

			br, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				h.log.Errorf("Could not read body of repeater response", err)
				return
			}

			// writte response to chan
			streader := bytes.NewReader(br)
			respBody, err := ioutil.ReadAll(streader)
			if err != nil {
				h.log.Errorf("Could not read body of repeater response", err)
			}
			stresp := NewStampedResponse(respBody, resp.Header, streq)
			h.Responses <- stresp
			h.store.addResponse(stresp)

		}
	}()
}

func (h *Handler) StartComparer() {
	go func() {
		for resp := range h.Responses {

			record := h.store.getRecord(resp.stamp)
			h.log.
				WithTransactionID(record.req.tid).
				Info("Received response")

			if len(record.responses) == 2 {

				b1 := string(record.responses[0].body)
				b2 := string(record.responses[1].body)
				if b1 != b2 {
					h.log.
						WithTransactionID(record.req.tid).
						Error("Different results were returned")
					continue
				}

				h.log.
					WithTransactionID(record.req.tid).
					Debug("results were idential")
			}
		}
	}()
}

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
				for _, r := range records {
					numResponses := len(r.responses)
					if numResponses == 2 {
						h.store.removeRecord(r.req.stamp)
						h.log.Debug("Removed old record with 2 responses")
						continue
					}
					if numResponses < 2 {
						h.store.removeRecord(r.req.stamp)
						req := r.req.req
						reqInfo := fmt.Sprintf("%s:%s%s", req.Method, req.Host, req.URL.String())
						h.log.Errorf("request %s received less than two responses; one of the services might be faulty", reqInfo)
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func copyRequest(r http.Request) (http.Request, error) {

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return r, err
	}
	bodycp := ioutil.NopCloser(strings.NewReader(string(buf)))

	return http.Request{
		Body:          bodycp,
		Method:        r.Method,
		Proto:         r.Proto,
		ProtoMajor:    r.ProtoMajor,
		ProtoMinor:    r.ProtoMinor,
		ContentLength: r.ContentLength,
		Header:        r.Header,
		Trailer:       r.Trailer,
		Host:          r.Host,
		URL: &url.URL{
			Scheme:     r.URL.Scheme,
			Opaque:     r.URL.Opaque,
			Host:       r.URL.Host,
			Path:       r.URL.Path,
			RawPath:    r.URL.RawPath,
			ForceQuery: r.URL.ForceQuery,
			RawQuery:   r.URL.RawQuery,
			Fragment:   r.URL.Fragment,
			User:       r.URL.User,
		},
	}, nil
}
