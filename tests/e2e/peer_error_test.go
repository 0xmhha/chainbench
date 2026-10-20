//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestAdminPeerCountDoesNotHideRPCFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ ID any }
		_ = json.NewDecoder(r.Body).Decode(&q)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "error": map[string]any{"code": -32000, "message": "peer backend unavailable"}})
	}))
	defer srv.Close()
	// Zero is a legitimate result; a transport/backend failure must stay distinguishable.
	count, err := adminPeerCount(rpc.Dial(srv.URL))
	var server *rpc.ServerError
	if count != 0 || !errors.As(err, &server) || server.Code != -32000 || server.Message != "peer backend unavailable" {
		t.Fatalf("lost RPC cause: count=%d err=%v", count, err)
	}
}
