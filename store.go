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

var newRecords chan (*Record)
var newResponses chan (*Record)

func init() {
	newRecords = make(chan *Record, 1000)
	newResponses = make(chan *Record, 1000)
}

type Response struct {
	body    []byte
	headers http.Header
}

type Record struct {
	req       http.Request
	responses []Response
	reqTime   int64
	tid       string
	stamp     string
}

func NewRecord(req http.Request) *Record {

	t := time.Now().UnixNano()
	stamp := fmt.Sprintf("%d:%s%s", t, req.Host, req.URL.String())
	hasher := md5.New()
	hasher.Write([]byte(stamp))
	hash := hex.EncodeToString(hasher.Sum(nil))

	tid := req.Header.Get("X-Request-Id")
	if tid == "" {
		tid = fmt.Sprintf("tid_%s", uniuri.NewLen(16))
	}

	return &Record{
		req:       req,
		responses: make([]Response, 0),
		reqTime:   t,
		tid:       tid,
		stamp:     hash,
	}
}

func (r *Record) addResponse(resp Response) {
	r.responses = append(r.responses, resp)

	newResponses <- r
}

type Store struct {
	data map[string]*Record
	mtx  *sync.Mutex
}

func NewStore() Store {
	s := Store{}
	s.data = make(map[string]*Record)
	s.mtx = &sync.Mutex{}
	return s
}

func (s *Store) NewRecord(req http.Request) *Record {
	rec := NewRecord(req)
	s.addRecord(rec)
	return rec
}

func (s *Store) addRecord(r *Record) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.data[r.stamp] = r

	newRecords <- r
}

func (s *Store) getRecord(hash string) *Record {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.data[hash]
}

func (s *Store) removeRecord(hash string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.data, hash)
}

func (s *Store) getRecordsOlderThan(t int64) []*Record {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	retval := make([]*Record, 0)
	for _, r := range s.data {
		if r.reqTime < t {
			retval = append(retval, r)
		}
	}

	return retval
}
