package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/websocket"
)

// Custom-error selectors and structured diagnostics must survive the typed
// RPC boundary; callers cannot distinguish two reverts from their text alone.
func TestServerErrorPreservesDataAcrossHTTPAndWebSocket(t *testing.T) {
	for _, transport := range []string{"http", "ws"} {
		for _, tc := range []struct {
			name    string
			data    any
			present bool
		}{
			{"custom error", "0xe49c14cb", true}, {"structured", map[string]any{"error": "0xe49c14cb"}, true}, {"number", float64(7), true}, {"null", nil, true}, {"missing", nil, false},
		} {
			t.Run(transport+"/"+tc.name, func(t *testing.T) {
				payload := map[string]any{"code": 3, "message": "execution reverted"}
				if tc.present {
					payload["data"] = tc.data
				}
				reply := map[string]any{"jsonrpc": "2.0", "id": 1, "error": payload}
				var err error
				if transport == "http" {
					s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(reply) }))
					defer s.Close()
					err = Dial(s.URL).Call(context.Background(), "eth_call", nil)
				} else {
					url := wsServer(t, func(c *websocket.Conn) { _ = c.WriteJSON(reply) })
					_, err = Subscribe(context.Background(), url, "newHeads")
				}
				var server *ServerError
				if !errors.As(err, &server) {
					t.Fatal("typed node error missing", err)
				}
				encoded, e := json.Marshal(server)
				if e != nil {
					t.Fatal(e)
				}
				fields := map[string]json.RawMessage{}
				if e = json.Unmarshal(encoded, &fields); e != nil {
					t.Fatal(e)
				}
				raw, present := fields["Data"]
				if present != tc.present {
					t.Fatalf("typed error lost wire data presence: got=%v want=%v", present, tc.present)
				}
				if present {
					var got any
					if e = json.Unmarshal(raw, &got); e != nil || !reflect.DeepEqual(got, tc.data) {
						t.Fatalf("typed error changed wire data: got=%v want=%v error=%v", got, tc.data, e)
					}
				}
			})
		}
	}
}
