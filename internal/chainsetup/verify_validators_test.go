package chainsetup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// validatorRPCServer answers the chain's getValidators JSON-RPC with a fixed
// set, so VerifyValidators can be driven without a real node.
func validatorRPCServer(t *testing.T, validators []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": req.ID, "result": validators,
		})
	}))
}

// wsForValidatorCheck builds a stablenet (wbft) workspace with one node pointed
// at srv, using the committed preset for the composed keys.
func wsForValidatorCheck(t *testing.T, srv *httptest.Server) (*Workspace, []string) {
	t.Helper()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	w, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Chain = "stablenet"
	w.state.KeysDir = presetDir
	w.state.Validators = 4
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", Host: u.Hostname(), Endpoints: node.Endpoints{HTTP: port}}}
	w.state.Target = resource.Spec{DataRoot: t.TempDir()}

	preset, err := store.LoadPreset(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	return w, preset.NetworkFor(4).Validators
}

func TestVerifyValidators_MatchAndMismatch(t *testing.T) {
	// Match: the chain reports exactly the composed validators.
	srvMatchWS := func(vals []string) (*Workspace, *httptest.Server) {
		srv := validatorRPCServer(t, vals)
		w, _ := wsForValidatorCheck(t, srv)
		return w, srv
	}

	// First learn the expected set from a throwaway workspace.
	probe := validatorRPCServer(t, nil)
	_, expected := wsForValidatorCheck(t, probe)
	probe.Close()

	// Match case.
	w, srv := srvMatchWS(expected)
	defer srv.Close()
	got, err := w.VerifyValidators(context.Background())
	if err != nil {
		t.Fatalf("VerifyValidators: %v", err)
	}
	if !got.Match {
		t.Fatalf("expected match; mismatch=%q actual=%v expected=%v", got.Mismatch, got.Actual, got.Expected)
	}
	if got.Method != "wbft runtime validators" {
		t.Fatalf("method = %q", got.Method)
	}

	// Mismatch case: the chain reports a different set.
	srv2 := validatorRPCServer(t, []string{"0x1111111111111111111111111111111111111111"})
	defer srv2.Close()
	w2, _ := wsForValidatorCheck(t, srv2)
	got2, err := w2.VerifyValidators(context.Background())
	if err != nil {
		t.Fatalf("VerifyValidators (mismatch): %v", err)
	}
	if got2.Match || got2.Mismatch == "" {
		t.Fatalf("expected a mismatch, got match=%v", got2.Match)
	}
}
