package engine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/engine"
)

func TestNewAttachEngine_Validation(t *testing.T) {
	if _, err := engine.NewAttachEngine(engine.AttachConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
	if _, err := engine.NewAttachEngine(engine.AttachConfig{Chain: "c", ArtifactRoot: t.TempDir()}); err == nil {
		t.Fatal("expected error for no RPC URLs")
	}
	if _, err := engine.NewAttachEngine(engine.AttachConfig{Chain: "c", ArtifactRoot: t.TempDir(), RPCURLs: []string{""}}); err == nil {
		t.Fatal("expected error for empty RPC URL")
	}
}

// TestAttachEngine_RunAgainstMockRPC runs a full attach-mode Engine.Run against
// a mock JSON-RPC node — no chain binary needed, so it runs in CI. It proves
// attach builds the node table from the endpoint and runs the spec's assertions,
// recording a pass.
func TestAttachEngine_RunAgainstMockRPC(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",  // 16
	})

	artifactRoot := t.TempDir()
	eng, err := engine.NewAttachEngine(engine.AttachConfig{
		Chain:        "stablenet",
		RPCURLs:      []string{srv.URL},
		ArtifactRoot: artifactRoot,
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewAttachEngine: %v", err)
	}

	spec, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "attach-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 1337},
			{"assert": "blockNumber", "expected": 1},
		},
	})

	root, err := eng.Run(context.Background(), [][]byte{spec})
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	var doc struct {
		Summary struct {
			Pass int `json:"pass"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse session.json: %v", err)
	}
	if doc.Summary.Pass < 1 {
		t.Fatalf("expected a passing test, got summary %+v", doc.Summary)
	}
}

func TestAttachEngine_SkipsInapplicable(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_chainId": "0x1"})
	eng, err := engine.NewAttachEngine(engine.AttachConfig{
		Chain:        "stablenet",
		RPCURLs:      []string{srv.URL},
		ArtifactRoot: t.TempDir(),
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewAttachEngine: %v", err)
	}
	// A spec that applies only to wbft must be skipped against a stablenet attach.
	spec, _ := json.Marshal(map[string]any{
		"schemaVersion":    "1",
		"id":               "wbft-only",
		"applicableChains": "wbft",
		"chain":            map[string]any{"name": "wbft", "binary": "go-wbft"},
		"assertions":       []map[string]any{{"assert": "chainId", "expected": 999}},
	})
	root, err := eng.Run(context.Background(), [][]byte{spec})
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "session.json"))
	var doc struct {
		Summary struct {
			Fail int `json:"fail"`
			Skip int `json:"skip"`
		} `json:"summary"`
	}
	_ = json.Unmarshal(data, &doc)
	// The failing assertion must NOT have run (skipped), so no fail is recorded.
	if doc.Summary.Fail != 0 {
		t.Fatalf("inapplicable spec should be skipped, not failed: %+v", doc.Summary)
	}
	if doc.Summary.Skip != 1 {
		t.Fatalf("expected 1 skipped test, got %+v", doc.Summary)
	}
}

// TestAttachEngine_SkipsUnmetCapability proves capability gating end to end: an
// attach network advertises only "rpc", so a spec requiring "ws" is skipped
// (its failing assertion never runs).
func TestAttachEngine_SkipsUnmetCapability(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_chainId": "0x1"})
	eng, err := engine.NewAttachEngine(engine.AttachConfig{
		Chain:        "stablenet",
		RPCURLs:      []string{srv.URL},
		ArtifactRoot: t.TempDir(),
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewAttachEngine: %v", err)
	}
	spec, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "needs-ws",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"requires":      []string{"ws"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 999}},
	})
	root, err := eng.Run(context.Background(), [][]byte{spec})
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "session.json"))
	var doc struct {
		Summary struct {
			Fail int `json:"fail"`
			Skip int `json:"skip"`
		} `json:"summary"`
	}
	_ = json.Unmarshal(data, &doc)
	if doc.Summary.Fail != 0 || doc.Summary.Skip != 1 {
		t.Fatalf("ws-requiring spec should skip on an rpc-only attach: %+v", doc.Summary)
	}
}

// TestAttachEngine_AdvertisesConfiguredCaps proves the operator-asserted Caps
// let an overlay-gated spec run: a net launched with the account-extra overlay
// carries a capability attach cannot detect over RPC, so naming it in Caps must
// make the gated spec run (its assertion executes and passes) rather than skip.
func TestAttachEngine_AdvertisesConfiguredCaps(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_chainId": "0x1"})
	eng, err := engine.NewAttachEngine(engine.AttachConfig{
		Chain:        "stablenet",
		RPCURLs:      []string{srv.URL},
		Caps:         []string{"account-extra"},
		ArtifactRoot: t.TempDir(),
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewAttachEngine: %v", err)
	}
	spec, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "needs-account-extra",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"requires":      []string{"rpc", "account-extra"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 1}},
	})
	root, err := eng.Run(context.Background(), [][]byte{spec})
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "session.json"))
	var doc struct {
		Summary struct {
			Pass int `json:"pass"`
			Skip int `json:"skip"`
		} `json:"summary"`
	}
	_ = json.Unmarshal(data, &doc)
	if doc.Summary.Pass != 1 || doc.Summary.Skip != 0 {
		t.Fatalf("account-extra spec should run when the cap is advertised: %+v", doc.Summary)
	}
}
