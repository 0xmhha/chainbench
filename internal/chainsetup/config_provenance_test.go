package chainsetup_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// readProvenance loads the config provenance the config step persisted to the
// workspace's composition file.
func readProvenance(t *testing.T, dir string) []chainsetup.ConfigProvenance {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "**", "workspace.json"))
	if err != nil || len(matches) == 0 {
		// The composition file may sit directly under dir.
		if _, statErr := os.Stat(filepath.Join(dir, "workspace.json")); statErr == nil {
			matches = []string{filepath.Join(dir, "workspace.json")}
		}
	}
	if len(matches) == 0 {
		t.Fatalf("no workspace.json under %s", dir)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	var state struct {
		ConfigProvenance []chainsetup.ConfigProvenance `json:"configProvenance"`
	}
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatalf("parse workspace.json: %v", err)
	}
	return state.ConfigProvenance
}

// TestNetConfig_OverrideIsolationAndProvenance pins E3: a node-scoped config
// override reaches only that node, the config the step wrote is recorded per
// node with its checksum, and the checksums differ when the configs do.
func TestNetConfig_OverrideIsolationAndProvenance(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	must := func(_ any, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(chainsetup.NetNew(ctx, d, chainsetup.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}))
	must(chainsetup.NetAllocate(ctx, d, chainsetup.NetAllocateIn{DataDir: dir, Validators: 3}))
	must(chainsetup.NetKeys(ctx, d, chainsetup.NetKeysIn{DataDir: dir}))
	must(chainsetup.NetGenesis(ctx, d, chainsetup.NetGenesisIn{DataDir: dir, ChainID: 9999}))
	// Every node gets metricsHost; only node2 gets httpHost.
	must(chainsetup.NetConfig(ctx, d, chainsetup.NetConfigIn{DataDir: dir, ScopedSet: map[string][]string{
		"all":   {"metricsHost=10.7.7.7"},
		"node2": {"httpHost=10.2.2.2"},
	}}))

	n2 := readConfig(t, dir, 2)
	n3 := readConfig(t, dir, 3)
	if !strings.Contains(n2, "10.7.7.7") || !strings.Contains(n3, "10.7.7.7") {
		t.Fatalf("all-scope override did not reach every node:\nn2=%s\nn3=%s", n2, n3)
	}
	if !strings.Contains(n2, "10.2.2.2") {
		t.Fatalf("node2's own override was not applied:\n%s", n2)
	}
	if strings.Contains(n3, "10.2.2.2") {
		t.Fatalf("node2's override leaked into node3's config:\n%s", n3)
	}

	prov := readProvenance(t, dir)
	if len(prov) != 3 {
		t.Fatalf("provenance entries = %d, want 3", len(prov))
	}
	byNode := map[int]chainsetup.ConfigProvenance{}
	for _, p := range prov {
		byNode[p.Node] = p
	}
	for _, n := range []int{1, 2, 3} {
		if !strings.HasPrefix(byNode[n].Checksum, "sha256:") {
			t.Errorf("node%d checksum missing or malformed: %q", n, byNode[n].Checksum)
		}
	}
	if !slices.Contains(byNode[2].Overrides, "httpHost=10.2.2.2") {
		t.Errorf("node2 provenance missing its override: %v", byNode[2].Overrides)
	}
	if slices.Contains(byNode[3].Overrides, "httpHost=10.2.2.2") {
		t.Errorf("node3 provenance carries node2's override: %v", byNode[3].Overrides)
	}
	if byNode[2].Checksum == byNode[3].Checksum {
		t.Error("node2 and node3 configs differ but their recorded checksums match")
	}
}
