package http

import (
	"net"
	"net/http"
	"time"
)

func NewHTTPClient() *http.Client {

	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}
}
