package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicPeersRejectMissingFields(t *testing.T) {
	for _, name := range []string{"valid", "last missing id", "last empty remote", "last missing network", "last missing protocols", "last empty protocols", "rpc error"} {
		t.Run(name, func(t *testing.T) {
			peers := []any{}
			for i := 2; i <= 5; i++ {
				peers = append(peers, map[string]any{"id": fmt.Sprint("id", i), "network": map[string]any{"remoteAddress": "127.0.0.1:30303"}, "protocols": map[string]any{"eth": map[string]any{"version": 68}}})
			}
			last := peers[3].(map[string]any)
			switch name {
			case "last missing id":
				delete(last, "id")
			case "last empty remote":
				last["network"].(map[string]any)["remoteAddress"] = ""
			case "last missing network":
				delete(last, "network")
			case "last missing protocols":
				delete(last, "protocols")
			case "last empty protocols":
				last["protocols"] = map[string]any{}
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				reply := map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": peers}
				if name == "rpc error" {
					delete(reply, "result")
					reply["error"] = map[string]any{"code": -32000, "message": "peer backend unavailable"}
				}
				if q.Method != "admin_peers" {
					t.Errorf("unexpected RPC %s", q.Method)
				}
				_ = json.NewEncoder(w).Encode(reply)
			}))
			defer srv.Close()
			raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/node/CT-NODE-008-peers.json"}, "../../presets/chain/stablenet-bp4-en1.json")
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Expect == "rpcCall" {
					delete(st.Args, "on")
					selected = append(selected, st)
				}
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"id2": "id2", "id3": "id3", "id4": "id4", "id5": "id5"}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			want := session.StatusFail
			if name == "valid" {
				want = session.StatusPass
			}
			if err != nil || got != want {
				t.Fatalf("public peers %s err=%v want=%s assertions=%+v", got, err, want, rec.asserts)
			}
		})
	}
}
