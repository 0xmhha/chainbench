package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRequiresUnambiguousCompleteResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		wantErr, node bool
	}{
		{"value", `{"result":"0xaccepted"}`, false, false},
		{"null value", `{"result":null}`, false, false},
		{"node rejection", `{"error":{"code":-32000,"message":"insufficient funds"}}`, true, true},
		{"legacy null with rejection", `{"result":null,"error":{"code":-32000,"message":"insufficient funds"}}`, true, true},
		{"explicit zero code empty message", `{"error":{"code":0,"message":""}}`, true, true},
		{"hash and rejection", `{"result":"0xaccepted","error":{"code":-32000,"message":"insufficient funds"}}`, true, false},
		{"object and rejection", `{"result":{"hash":"0xaccepted"},"error":{"code":-32000,"message":"insufficient funds"}}`, true, false},
		{"missing error code", `{"error":{"message":"insufficient funds"}}`, true, false},
		{"null error code", `{"error":{"code":null,"message":"insufficient funds"}}`, true, false},
		{"missing error message", `{"error":{"code":-32000}}`, true, false},
		{"null error message", `{"error":{"code":-32000,"message":null}}`, true, false},
		{"empty envelope", `{}`, true, false},
		{"null error without result", `{"error":null}`, true, false},
		{"second JSON after rejection", `{"error":{"code":-32000,"message":"insufficient funds"}} {"result":"0xaccepted"}`, true, false},
		{"garbage after rejection", `{"error":{"code":-32000,"message":"insufficient funds"}} trailing`, true, false},
		{"second JSON after value", `{"result":"0xaccepted"} {"result":"0xdifferent"}`, true, false},
	} {
		for _, discard := range []bool{false, true} {
			name := tc.name
			if discard {
				name += "/discard result"
			}
			t.Run(name, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(tc.body))
				}))
				defer srv.Close()
				var result any
				var output any = &result
				if discard {
					output = nil
				}
				err := Dial(srv.URL).Call(context.Background(), "eth_sendRawTransaction", output, "0x02")
				var se *ServerError
				if (err != nil) != tc.wantErr || errors.As(err, &se) != tc.node {
					t.Fatalf("err=%v nodeError=%v; want err=%v nodeError=%v", err, errors.As(err, &se), tc.wantErr, tc.node)
				}
				if tc.wantErr && result != nil {
					t.Fatal("failed response changed output", result)
				}
			})
		}
	}
}
