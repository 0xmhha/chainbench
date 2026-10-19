package testhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicInvalidP256RequiresExactEmptyResult(t *testing.T) {
	raw, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json"}, "../../presets/chain/wbft-bp4.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		value any
		pass  bool
	}{
		{"exact empty", "0x", true}, {"zero word", "0x" + strings.Repeat("0", 64), false}, {"garbage hex", "0x1234", false}, {"malformed string", "invalid", false}, {"object", map[string]any{"value": "0x"}, false}, {"null", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec, e := dsl.ParseV2(raw[0])
			if e != nil {
				t.Fatal(e)
			}
			server := mockRPC(t, map[string]any{"eth_call": tc.value})
			rec := &recordStub{}
			status, e := interp.NewInterpreter(deps()).Run(context.Background(), spec, envWithNode(t, server.URL), rec)
			if tc.pass {
				if e != nil || status != session.StatusPass {
					t.Fatalf("exact empty rejected: %s %v %+v", status, e, rec.asserts)
				}
			} else if e == nil && status == session.StatusPass {
				t.Fatalf("invalid output accepted: %+v", rec.asserts)
			}
		})
	}
}

// Exercise the public bytecode checkpoint with adversarial RPC responses.
// Activation timing/P256/balance behavior remain in the full native case.
func TestPublicDelayedForkRequiresUsablePostCode(t *testing.T) {
	raw, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	valid := "0x" + strings.Repeat("61", 50)
	for _, tc := range []struct {
		name      string
		pre, post any
		pass      bool
	}{
		{"minimum valid transition", "0x60", valid, true},
		{"empty post", "0x60", "0x", false}, {"short post", "0x60", "0x61", false}, {"49 bytes", "0x60", "0x" + strings.Repeat("61", 49), false},
		{"odd post hex", "0x60", valid + "a", false}, {"nonhex post", "0x60", "0x" + strings.Repeat("zz", 50), false}, {"object post", "0x60", map[string]any{"code": valid}, false}, {"number post", "0x60", 123, false},
		{"unchanged code", valid, valid, false}, {"bad pre hex", "0xzz", valid, false}, {"object pre", map[string]any{"code": "0x60"}, valid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec, e := dsl.ParseV2(raw[0])
			if e != nil {
				t.Fatal(e)
			}
			code := []dsl.Statement{}
			for _, st := range spec.Sequence {
				if st.Args["method"] == "eth_call" {
					break
				}
				if st.Do != "waitBlock" {
					code = append(code, st)
				}
			}
			spec.Sequence = code
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if e := json.NewDecoder(r.Body).Decode(&q); e != nil {
					t.Error(e)
					return
				}
				if q.Method != "eth_getCode" || len(q.Params) != 2 {
					t.Errorf("unexpected request %+v", q)
					http.Error(w, "unexpected", 400)
					return
				}
				value := tc.post
				if q.Params[1] == "0x1" {
					value = tc.pre
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": value})
			}))
			defer srv.Close()
			rec := &recordStub{}
			status, e := interp.NewInterpreter(deps()).Run(context.Background(), spec, envWithNode(t, srv.URL), rec)
			if tc.pass {
				if e != nil || status != session.StatusPass {
					t.Fatalf("valid transition rejected: %s %v %+v", status, e, rec.asserts)
				}
			} else if e == nil && status == session.StatusPass {
				t.Fatalf("bad code transition accepted: %+v", rec.asserts)
			}
		})
	}
}
