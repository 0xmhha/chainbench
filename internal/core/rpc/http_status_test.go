package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPFailureCannotBecomeNodeRejection(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"insufficient funds"}}`))
			}))
			defer server.Close()
			err := Dial(server.URL).Call(context.Background(), "eth_sendRawTransaction", nil, "0x00")
			var rejected *ServerError
			if err == nil || errors.As(err, &rejected) {
				t.Fatalf("HTTP %d masqueraded as a node rejection: %v", status, err)
			}
		})
	}
}
