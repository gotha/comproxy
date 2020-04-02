package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dchest/uniuri"
)

type StampedRequest struct {
	req          http.Request
	stamp        string
	timeReceived int64
	tid          string
}

func NewStampedRequest(r http.Request) StampedRequest {
	t := time.Now().UnixNano()
	stamp := fmt.Sprintf("%d:%s%s", t, r.Host, r.URL.String())
	hasher := md5.New()
	hasher.Write([]byte(stamp))
	hash := hex.EncodeToString(hasher.Sum(nil))

	tid := r.Header.Get("X-Request-Id")
	if tid == "" {
		tid = fmt.Sprintf("tid_%s", uniuri.NewLen(16))
	}

	return StampedRequest{
		req:          r,
		stamp:        hash,
		timeReceived: t,
		tid:          tid,
	}
}

func (sr *StampedRequest) GetDescr() string {
	return fmt.Sprintf("%s:%s%s", sr.req.Method, sr.req.Host, sr.req.URL.String())
}

func (sr *StampedRequest) GetProperties() map[string]interface{} {
	return map[string]interface{}{
		"method": sr.req.Method,
		"host":   sr.req.Host,
		"uri":    sr.req.URL.String(),
	}
}

type StampedResponse struct {
	body    []byte
	headers http.Header
	stamp   string
}

func NewStampedResponse(b []byte, h http.Header, req StampedRequest) StampedResponse {
	return StampedResponse{
		body:    b,
		headers: h,
		stamp:   req.stamp,
	}
}

type Record struct {
	req       StampedRequest
	responses []StampedResponse
}

type Store struct {
	data map[string]Record
	mtx  *sync.Mutex
}

func NewStore() Store {
	s := Store{}
	s.data = make(map[string]Record)
	s.mtx = &sync.Mutex{}
	return s
}

func (s *Store) addRequest(sr StampedRequest) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.data[sr.stamp] = Record{
		req: sr,
	}
}

func (s *Store) addResponse(sr StampedResponse) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	record := s.data[sr.stamp]
	record.responses = append(record.responses, sr)
	s.data[sr.stamp] = record
}

func (s *Store) removeRecord(stamp string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.data, stamp)
}

func (s *Store) getRecord(stamp string) Record {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.data[stamp]
}

func (s *Store) getRecordsOlderThan(t int64) []Record {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	retval := make([]Record, 0)
	for _, r := range s.data {
		if r.req.timeReceived < t {
			retval = append(retval, r)
		}
	}

	return retval
}
