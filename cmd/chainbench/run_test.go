package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

// mockRPCNode serves canned JSON-RPC results keyed by method.
func mockRPCNode(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if v, ok := results[req.Method]; ok {
			res["result"] = v
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeSpec(t *testing.T, spec map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(spec)
	p := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunCmd_AttachMockRPC(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",  // 16
	})
	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "cli-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 1337},
			{"assert": "blockNumber", "expected": 1},
		},
	})

	out, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), specPath)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pass=1") {
		t.Fatalf("expected pass=1 in output:\n%s", out)
	}
	if !strings.Contains(out, "cli-smoke") {
		t.Fatalf("expected the test id in output:\n%s", out)
	}
}

func TestRunCmd_FailingSpecExitsNonZero(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{"eth_chainId": "0x1"}) // chainId 1
	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "cli-fail",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 999}},
	})
	out, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), specPath)
	if err == nil {
		t.Fatalf("expected a non-nil error for a failing spec:\n%s", out)
	}
	if !strings.Contains(out, "fail=1") {
		t.Fatalf("expected fail=1 in output:\n%s", out)
	}
}

// TestRunCmd_DashboardStreamsEvents proves the full T6.3 path end-to-end in CI:
// the engine emits orchestration events → local bus → dashboard.Forward → a
// running chainbenchd (dashboard.Server) → its bus. It uses attach mode against
// a mock RPC node, so no chain binary is needed.
func TestRunCmd_DashboardStreamsEvents(t *testing.T) {
	rpc := mockRPCNode(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",
	})

	// A chainbenchd whose bus we can observe.
	srvBus := obs.NewBus()
	defer srvBus.Close()
	sub := srvBus.Subscribe()
	dsrv := httptest.NewServer(dashboard.NewServer(srvBus, nil))
	defer dsrv.Close()

	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "dash-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 1337}},
	})

	out, err := run(t, "run", "--chain", "stablenet", "--rpc", rpc.URL,
		"--dashboard", dsrv.URL, "--artifact-root", t.TempDir(), specPath)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	// The CLI flushes the forwarder before returning, so every event is already
	// POSTed and published to srvBus. Drain what the server received.
	got := map[string]bool{}
	for draining := true; draining; {
		select {
		case e := <-sub:
			got[e.Message] = true
		default:
			draining = false
		}
	}
	for _, want := range []string{"run started", "running spec", "spec pass", "run complete"} {
		if !got[want] {
			t.Fatalf("dashboard did not receive %q; got %v", want, got)
		}
	}
}

func TestRunCmd_RequiresMode(t *testing.T) {
	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "x",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "True"}},
	})
	// Neither --rpc nor --binary given.
	if _, err := run(t, "run", "--chain", "stablenet", specPath); err == nil {
		t.Fatal("expected error when neither attach nor local mode is selected")
	}
	// No --chain.
	if _, err := run(t, "run", "--rpc", "http://127.0.0.1:1", specPath); err == nil {
		t.Fatal("expected error when --chain is missing")
	}
}

func TestRunCmd_ExitCodes(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{"eth_chainId": "0x539", "eth_blockNumber": "0x10"})

	// pass -> 0
	passSpec := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "ok",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "chainId", "expected": 1337}},
	})
	if _, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), passSpec); exitCode(err) != 0 {
		t.Fatalf("pass exit = %d, want 0 (err=%v)", exitCode(err), err)
	}

	// fail -> 1
	failSpec := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "bad",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "chainId", "expected": 999}},
	})
	if _, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), failSpec); exitCode(err) != 1 {
		t.Fatalf("fail exit = %d, want 1 (err=%v)", exitCode(err), err)
	}

	// blocked (malformed spec) -> 2
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), bad); exitCode(err) != 2 {
		t.Fatalf("blocked exit = %d, want 2 (err=%v)", exitCode(err), err)
	}
}

func TestRunCmd_JSONOutput(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{"eth_chainId": "0x539", "eth_blockNumber": "0x10"})
	spec := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "json-smoke",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "chainId", "expected": 1337}},
	})
	out, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), "--json", spec)
	if err != nil {
		t.Fatalf("run --json: %v\n%s", err, out)
	}
	var rep struct {
		Session string `json:"session"`
		Tests   []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tests"`
		Summary struct{ Pass int } `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if rep.Summary.Pass != 1 || len(rep.Tests) != 1 || rep.Tests[0].ID != "json-smoke" || rep.Session == "" {
		t.Fatalf("unexpected JSON report: %+v", rep)
	}
}

func TestKeySource_FlagMapping(t *testing.T) {
	cases := []struct {
		name    string
		opts    runOpts
		want    string
		wantErr string
	}{
		{
			name: "default is the reproducible preset",
			opts: runOpts{keysDir: "keys/preset"},
			want: "preset:keys/preset",
		},
		{
			name: "explicit preset",
			opts: runOpts{keysDir: "k", keysSource: "preset"},
			want: "preset:k",
		},
		{
			name: "generate needs a bootnode for BLS material",
			opts: runOpts{keysDir: "k", keysSource: "generate"},
			// Silently generating a set without BLS keys would produce a chain
			// that fails much later, so this must be refused up front.
			wantErr: "--bootnode",
		},
		{
			name: "generate with a bootnode",
			opts: runOpts{keysDir: "k", keysSource: "generate", bootnode: "/bin/bootnode"},
			want: "generated:k",
		},
		{
			name:    "unknown source is rejected",
			opts:    runOpts{keysDir: "k", keysSource: "borrow"},
			wantErr: "unknown --keys-source",
		},
		{
			name:    "a local run needs a key directory",
			opts:    runOpts{keysSource: "preset"},
			wantErr: "--keys is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src, err := keySource(tc.opts)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want an error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("keySource: %v", err)
			}
			if got := src.Describe(); got != tc.want {
				t.Errorf("Describe() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestParseLaunchOverrides(t *testing.T) {
	got, err := parseLaunchOverrides([]string{"networkid=4242", "nodiscover", "http.api=eth,net"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("overrides = %d, want 3", len(got))
	}
	if got[0].Key != launchopt.KeyNetworkID || got[0].Value != "4242" {
		t.Errorf("override[0] = %+v", got[0])
	}
	if got[1].Key != launchopt.KeyNoDiscover || got[1].Value != "" {
		t.Errorf("bare boolean key: %+v", got[1])
	}

	if _, err := parseLaunchOverrides([]string{"=oops"}); err == nil {
		t.Fatal("empty key must be rejected at the CLI boundary")
	}
}
