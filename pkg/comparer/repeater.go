package comparer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

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
func StartRepeater(URL *url.URL) {

	go func() {
		for rr := range bridgeChan {

			fmt.Printf("I have to repeat request: %+v\n", rr.tid)

			// create new request
			body := rr.req.Context().Value(proxy.ReqBodyKey).(string)
			bodyReader := bytes.NewReader([]byte(body))
			req, err := http.NewRequest(rr.req.Method, rr.req.URL.String(), bodyReader)
			if err != nil {
				fmt.Printf("%+v\n", err)
				panic("We cannot repeat the request")
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

				fmt.Printf("%+v\n", err)
				panic("HOI")
				//rw.WriteHeader(http.StatusInternalServerError)
				//fmt.Fprint(rw, err)
				//return
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

			repeaterChan <- rrs
		}
	}()

}
