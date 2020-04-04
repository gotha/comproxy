package comparer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	httpclient "github.com/gotha/comproxy/pkg/http"
	"github.com/jinzhu/copier"
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

			fmt.Printf("%+v\n", "I have to repeat request")
			fmt.Printf("%+v\n", rr.tid)

			req, err := http.NewRequest(rr.req.Method, rr.req.URL.String())

			var req http.Request
			copier.Copy(&req, &rr.req)

			// modify request to go to new target
			req.Host = URL.Host
			req.URL.Host = URL.Host
			req.URL.Scheme = URL.Scheme
			req.RequestURI = ""

			httpClient := httpclient.NewHTTPClient()
			resp, err := httpClient.Do(&req)
			if err != nil {

				fmt.Printf("%+v\n", err)
				panic("HOI")
				//rw.WriteHeader(http.StatusInternalServerError)
				//fmt.Fprint(rw, err)
				//return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)

			rrs := ReqResponses{
				tid: rr.tid,
				req: rr.req,
				responses: []Response{
					rr.response,
					Response{
						body:       string(body),
						statusCode: resp.StatusCode,
					},
				},
			}

			repeaterChan <- rrs
		}
	}()

}
