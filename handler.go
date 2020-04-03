package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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
	log      *logger.UPPLogger
	Services Services
	store    Store
}

func NewHandler(s Services, log *logger.UPPLogger) Handler {
	return Handler{
		Services: s,
		log:      log,
		store:    NewStore(),
	}
}

func (h *Handler) GetProxy() (http.HandlerFunc, error) {

	URL, err := url.Parse(h.Services.Primary.url)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		rec := h.store.NewRecord(*req)

		// @todo - log body if it is not empty
		h.log.
			WithField("method", req.Method).
			WithField("url", req.URL.String()).
			WithTransactionID(rec.tid).
			Info("Received request")

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
		defer resp.Body.Close()

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

		var bufBody bytes.Buffer
		body := io.TeeReader(resp.Body, &bufBody)
		_, err = io.Copy(rw, body)
		if err != nil {

		}

		// copy trailer
		for k, values := range resp.Trailer {
			for _, v := range values {
				rw.Header().Set(k, v)
			}
		}

		// add response to record
		rec.addResponse(Response{
			body: bufBody.Bytes(),
		})

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
		for rec := range newRecords {

			r, err := copyRequest(rec.req)
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
			defer resp.Body.Close()

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
				body: respBody,
			})

		}
	}()
}

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
