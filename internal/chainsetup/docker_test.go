package chainsetup_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// TestWorkspace_DockerModePersists pins that the mode is recorded once at
// `new` and survives a reopen — the reason it lives in the workspace is that
// requiring the flag on every step would allow a half-mapped run.
func TestWorkspace_DockerModePersists(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", Docker: true}); err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}

	reopened, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.State().Docker {
		t.Fatal("docker mode was not persisted")
	}
}

// TestWorkspace_DockerWithoutLocalmapRefusesLoudly pins the activation rule:
// the flag without the mapping file is an error naming the fix, not a silent
// unmapped dial. Health is the first step that resolves an address, so it is
// where the refusal must surface.
func TestWorkspace_DockerWithoutLocalmapRefusesLoudly(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", Docker: true}); err != nil {
		t.Fatalf("New: %v", err)
	}
	// A minimal node table so Health reaches the address resolution.
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Validators: 1}); err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	_, err = ws.Health(context.Background())
	if err == nil {
		t.Fatal("docker mode without a localmap should refuse")
	}
	if !strings.Contains(err.Error(), "--docker") {
		t.Fatalf("the refusal should name the option that demands the file: %v", err)
	}
}

// TestNodeSet_MetricsURLIsTranslatedLikeTheRPCURL pins the fix for the reason
// the metric assertion never worked under --docker: Host and Ports hold the
// node's OWN address, which is what peers dial and what lands in the genesis,
// and the harness reaches a container at its published loopback port instead.
// RPCURL has always carried that translation; the metrics endpoint did not, so
// the assertion composed Host+Ports by hand and timed out against an address
// only the container can reach.
func TestNodeSet_MetricsURLIsTranslatedLikeTheRPCURL(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet"}); err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Validators: 1}); err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	ns := ws.NodeSet()
	if len(ns.Nodes) != 1 {
		t.Fatalf("got %d nodes", len(ns.Nodes))
	}
	n := ns.Nodes[0]
	if n.Ports.Metrics == 0 {
		t.Fatal("placement assigned no metrics port, so there is nothing to reach")
	}
	if n.MetricsURL == "" {
		t.Fatal("the composition recorded no metrics endpoint; the assertion would have to build one from Host+Ports, which is the bug")
	}
	// Without docker the two agree — the translation is identity, and that is
	// what makes the recorded endpoint safe to prefer unconditionally.
	want := fmt.Sprintf("http://%s:%d", n.Host, n.Ports.Metrics)
	if n.MetricsURL != want {
		t.Errorf("MetricsURL = %q, want %q for an untranslated target", n.MetricsURL, want)
	}
	if n.RPCURL == n.MetricsURL {
		t.Errorf("the metrics endpoint must not be the RPC one: %s", n.MetricsURL)
	}
}
