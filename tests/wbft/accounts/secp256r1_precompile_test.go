package accounts_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

const (
	p256Precompile = "0x0000000000000000000000000000000000000100"
	p256ValidData  = "0x" +
		"4fc05e7c6fa9dcfc4d09b26e3352487de9f248699f930af99fc652c7374dc89c" +
		"c1e7730e0dd29f6e231649b948bf9dfa73b48f94e904615c3a6573fe05c3b6bf" +
		"68781757cc58563fb2ea9243822fcb3b973461be164b8d57f986146d16f2ee5e" +
		"1ccbe91c075fc7f4f033bfa248db8fccd3565de94bbfb12f3c59ff46c271bf83" +
		"ce4014c68811f9a21a1fdb2c0e6113e06db7ca93b7404e78dc7ccd5ca89a4ca9"
	p256One = "0x0000000000000000000000000000000000000000000000000000000000000001"
)

// p256Mock is an eth_call node that emulates the RIP-7212 precompile: it returns
// 0x..01 only for the exact valid 160-byte vector, and empty ("0x") otherwise —
// the same shape geth produces for a verification failure or a bad-length input.
func p256Mock(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int           `json:"id"`
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		result := "0x1"
		if req.Method == "eth_call" {
			result = "0x"
			if len(req.Params) > 0 {
				if call, ok := req.Params[0].(map[string]interface{}); ok {
					if call["to"] == p256Precompile && call["data"] == p256ValidData {
						result = p256One
					}
				}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
}

func runP256Case(t *testing.T, name string) testkit.Result {
	t.Helper()
	srv := p256Mock(t)
	t.Cleanup(srv.Close)

	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 {
		t.Fatalf("expected 1 result for %s, got %d", name, len(rep.Results))
	}
	return rep.Results[0]
}

func TestSecp256r1Cases_RegisterAndPass(t *testing.T) {
	want := map[string]bool{
		"secp256r1-precompile-valid":       false,
		"secp256r1-precompile-invalid":     false,
		"secp256r1-precompile-short-input": false,
	}
	for _, c := range testkit.Cases() {
		if _, ok := want[c.Name]; ok {
			want[c.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("case %q not registered on import", name)
		}
	}

	for name := range want {
		if got := runP256Case(t, name); got.Status != testkit.StatusPass {
			t.Fatalf("%s: status %s (%s)", name, got.Status, got.Message)
		}
	}
}

// TestSecp256r1Valid_FailsWhenPrecompileMissing confirms the valid case fails
// (not silently passes) when the node returns empty output — i.e. the precompile
// is absent or broken.
func TestSecp256r1Valid_FailsWhenPrecompileMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		result := "0x1"
		if req.Method == "eth_call" {
			result = "0x" // precompile returns nothing
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer srv.Close()

	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"secp256r1-precompile-valid"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusFail {
		t.Fatalf("expected fail when precompile missing, got %+v", rep.Results)
	}
}
