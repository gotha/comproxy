package comparer

import (
	"net/http"

	"github.com/gotha/comproxy/pkg/proxy"
)

var bridgeChan (chan ReqResponse)

func init() {
	bridgeChan = make(chan ReqResponse, 1000)
}

// Response - contains data about response to request
type Response struct {
	body        string
	respHeaders http.Header
	statusCode  int
}

// ReqResponse - contains data about request and response for single transaction
type ReqResponse struct {
	tid      string
	req      http.Request
	response Response
}

// Bridge - middleware that puts all received request and responses to channel
func Bridge(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		tid := req.Header.Get(proxy.TransactionIDGHeader)
		body := w.(proxy.ResponseWriterWithValue).Val.String()
		statusCode := w.(proxy.ResponseWriterWithValue).StatusCode

		bridgeChan <- ReqResponse{
			tid: tid,
			req: *req,
			response: Response{
				body:       body,
				statusCode: statusCode,
			},
		}

		next.ServeHTTP(w, req)
	})
}
