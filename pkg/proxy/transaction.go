package proxy

import (
	"fmt"
	"net/http"

	"github.com/dchest/uniuri"
)

// TransactionIDGHeader - key for header used to store transaction ID
const TransactionIDGHeader = "X-Request-Id"

// TransactionHandler - adds transaction_id if missing
func TransactionHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		tid := req.Header.Get(TransactionIDGHeader)
		if tid == "" {
			tid = fmt.Sprintf("tid_%s", uniuri.NewLen(16))
			req.Header.Set(TransactionIDGHeader, tid)
		}

		next.ServeHTTP(rw, req)

	})
}
