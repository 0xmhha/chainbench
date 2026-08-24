package netcompose_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/netcompose"
)

// TestWorkspace_DockerModePersists pins that the mode is recorded once at
// `new` and survives a reopen — the reason it lives in the workspace is that
// requiring the flag on every step would allow a half-mapped run.
func TestWorkspace_DockerModePersists(t *testing.T) {
	dir := t.TempDir()
	ws, err := netcompose.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(netcompose.NewOpts{Chain: "stablenet", Docker: true}); err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}

	reopened, err := netcompose.Open(dir, fixedClock())
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
	ws, err := netcompose.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(netcompose.NewOpts{Chain: "stablenet", Docker: true}); err != nil {
		t.Fatalf("New: %v", err)
	}
	// A minimal node table so Health reaches the address resolution.
	if _, err := ws.Allocate(netcompose.AllocateOpts{Validators: 1}); err != nil {
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
