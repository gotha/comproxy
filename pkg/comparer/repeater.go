package comparer

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Financial-Times/go-logger/v2"
	httpclient "github.com/gotha/comproxy/pkg/http"
	"github.com/gotha/comproxy/pkg/proxy"
)

var repeaterChan (chan ReqResponses)

// ReqResponses - struct that contains single request and multiple responses to it
type ReqResponses struct {
	tid       string
	req       http.Request
	responses []Response
}

func init() {
	repeaterChan = make(chan ReqResponses, 1000)
}

// StartRepeater - starts a goroutine that processes requests from bridgeChan
func StartRepeater(URL *url.URL, log *logger.UPPLogger, logBody bool) {

	go func() {
		for rr := range bridgeChan {

			log.WithTransactionID(rr.tid).Debug("Repeating request")

			// create new request
			body := rr.req.Context().Value(proxy.ReqBodyKey).(string)
			bodyReader := bytes.NewReader([]byte(body))
			req, err := http.NewRequest(rr.req.Method, rr.req.URL.String(), bodyReader)
			if err != nil {
				log.WithTransactionID(rr.tid).Errorf("Unable to create repeat request")
				continue
			}

			req.Header = rr.req.Header

			// modify request to go to new target
			req.Host = URL.Host
			req.URL.Host = URL.Host
			req.URL.Scheme = URL.Scheme
			req.RequestURI = ""
			// @todo - check if I am not missing to copy something

			// make request
			httpClient := httpclient.NewHTTPClient()
			resp, err := httpClient.Do(req)
			if err != nil {
				log.WithTransactionID(rr.tid).Errorf("Unable to execute repeat request")
				continue
			}
			defer resp.Body.Close()

			respBody, err := ioutil.ReadAll(resp.Body)

			rrs := ReqResponses{
				tid: rr.tid,
				req: rr.req,
				responses: []Response{
					rr.response,
					Response{
						body:       string(respBody),
						statusCode: resp.StatusCode,
					},
				},
			}

			var logRespBody string
			if logBody == true {
				logRespBody = string(respBody)
			}
			log.
				WithField("method", req.Method).
				WithField("url", req.URL.String()).
				WithField("body", logRespBody).
				WithTransactionID(rr.tid).
				Info("Returned response")

			repeaterChan <- rrs
		}
	}()

}
