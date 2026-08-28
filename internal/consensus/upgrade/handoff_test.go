package upgrade_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// presetPath is the repository's shipped key set.
func presetPath() string { return filepath.Join("..", "..", "..", "keys", "preset") }

// handoffInputs are the golden profile and preset over a temp data dir, with
// a runner that answers the base-genesis step by writing the fixture the
// binary would have produced.
func handoffInputs(t *testing.T) upgrade.HandoffInputs {
	t.Helper()
	dataDir := t.TempDir()
	exec := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		for i, a := range args {
			if a == "--out" && i+1 < len(args) {
				return nil, os.WriteFile(args[i+1], []byte(fromGenesis), 0o644)
			}
		}
		return nil, nil
	}
	return upgrade.HandoffInputs{
		ProfilePath: goldenProfilePath(), PresetDir: presetPath(),
		FromBinary: "gwemix", ToBinary: "gwbft", Template: "template.json",
		DataDir: dataDir, Exec: exec,
	}
}

// TestHandoff_ComposesFromProfileAndPreset walks the paper half of the
// sequence — config, base genesis, plan, overlay — and checks each step left
// what the next one reads: the governance config names the producer from the
// preset, the plan has one node per profile role with a pubkey each, and the
// overlay reaches the plan's genesis.
func TestHandoff_ComposesFromProfileAndPreset(t *testing.T) {
	in := handoffInputs(t)
	h, err := upgrade.NewHandoff(in)
	if err != nil {
		t.Fatalf("NewHandoff: %v", err)
	}
	if !strings.Contains(h.Describe(), "wemix -> wbft") {
		t.Fatalf("Describe = %q", h.Describe())
	}

	cfgPath, err := h.WriteConfig(context.Background())
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	var cfg poa.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("config is not a poa config: %v", err)
	}
	if len(cfg.Members) != 1 || !cfg.Members[0].Bootnode || cfg.Members[0].Addr != h.ProducerAccount() {
		t.Fatalf("config members = %+v, want the one producer as bootnode", cfg.Members)
	}
	if len(cfg.Accounts) != 1+len(h.Preset.NetworkFor(0).Validators) {
		t.Fatalf("alloc funds %d accounts, want producer + %d validators", len(cfg.Accounts), len(h.Preset.NetworkFor(0).Validators))
	}

	basePath, err := h.BaseGenesis(context.Background())
	if err != nil {
		t.Fatalf("BaseGenesis: %v", err)
	}
	if err := h.ComposePlan(basePath); err != nil {
		t.Fatalf("ComposePlan: %v", err)
	}
	want := h.Profile.Roles.Producers + h.Profile.Roles.Validators
	if len(h.Plan.Nodes) != want {
		t.Fatalf("plan has %d nodes, want %d", len(h.Plan.Nodes), want)
	}
	if enodes := h.Plan.Enodes(""); len(enodes) != want {
		t.Fatalf("plan yields %d enodes, want one per node (pubkeys from the preset)", len(enodes))
	}
	if !h.Plan.Nodes[0].Producer {
		t.Fatal("the first plan node must be the producer")
	}
	if ipc := h.ProducerIPC(node.Node{Index: 0}); !strings.HasSuffix(ipc, filepath.Join("node1", "gwemix.ipc")) {
		t.Fatalf("producer IPC = %q, want node1/gwemix.ipc under the data dir", ipc)
	}
}

func TestHandoff_ApplyOverlayReachesTheGenesis(t *testing.T) {
	in := handoffInputs(t)
	overlay := filepath.Join(t.TempDir(), "overlay.json")
	if err := os.WriteFile(overlay, []byte(`{"genesis":{"config":{"overlayMarker":7}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	in.GenesisOverlay = overlay
	h, err := upgrade.NewHandoff(in)
	if err != nil {
		t.Fatalf("NewHandoff: %v", err)
	}
	if _, err := h.WriteConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	basePath, err := h.BaseGenesis(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := h.ComposePlan(basePath); err != nil {
		t.Fatal(err)
	}
	detail, err := h.ApplyOverlay()
	if err != nil {
		t.Fatalf("ApplyOverlay: %v", err)
	}
	if !strings.Contains(detail, "merged") || !strings.Contains(string(h.Plan.Genesis), `"overlayMarker"`) {
		t.Fatalf("overlay not applied: detail=%q", detail)
	}
}

func TestNewHandoff_RejectsMissingInputs(t *testing.T) {
	good := handoffInputs(t)
	cases := []struct {
		name string
		edit func(*upgrade.HandoffInputs)
	}{
		{"no profile", func(in *upgrade.HandoffInputs) { in.ProfilePath = "" }},
		{"no template", func(in *upgrade.HandoffInputs) { in.Template = "" }},
		{"no from binary", func(in *upgrade.HandoffInputs) { in.FromBinary = "" }},
		{"no data dir", func(in *upgrade.HandoffInputs) { in.DataDir = "" }},
		{"unknown preset", func(in *upgrade.HandoffInputs) { in.PresetDir = t.TempDir() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := good
			tc.edit(&in)
			if _, err := upgrade.NewHandoff(in); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// TestHandoff_StepsNeedTheirPredecessors: the sequence is explicit, and a
// step run out of order says so instead of working on nothing.
func TestHandoff_StepsNeedTheirPredecessors(t *testing.T) {
	h, err := upgrade.NewHandoff(handoffInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.BaseGenesis(context.Background()); err == nil {
		t.Fatal("BaseGenesis before WriteConfig should fail")
	}
	if _, err := h.Launch(context.Background()); err == nil {
		t.Fatal("Launch before ComposePlan should fail")
	}
}
