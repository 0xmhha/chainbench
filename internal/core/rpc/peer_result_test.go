package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPeerMutationRequiresTrueResult(t *testing.T) {
	for _, method := range []string{"admin_addPeer", "admin_removePeer"} {
		for _, accepted := range []bool{false, true} {
			t.Run(method+map[bool]string{false: " refused", true: " accepted"}[accepted], func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var q struct {
						ID     any
						Method string
					}
					_ = json.NewDecoder(r.Body).Decode(&q)
					if q.Method != method {
						t.Errorf("method %s", q.Method)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": accepted})
				}))
				defer srv.Close()
				c := Dial(srv.URL)
				var err error
				if method == "admin_addPeer" {
					err = c.AddPeer(context.Background(), "enode://fixture")
				} else {
					err = c.RemovePeer(context.Background(), "enode://fixture")
				}
				if accepted && err != nil || !accepted && (err == nil || !strings.Contains(err.Error(), method)) {
					t.Fatalf("accepted=%v err=%v", accepted, err)
				}
			})
		}
	}
}
