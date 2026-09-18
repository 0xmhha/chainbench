package testengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteOverlay_TwoOverlaysGetTwoFiles is the property one fixed filename
// could not have. Several specs run in one workspace, so with a single name the
// second overlay overwrote the first — and the earlier environment's recorded
// request then pointed at the later one's bytes, which is the one place a
// composition's own record could not be trusted. Preflight, which can only see
// what the request names, then could not tell the two chains apart.
func TestWriteOverlay_TwoOverlaysGetTwoFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	applepie, err := writeOverlay(ctx, dir, map[string]any{"config": map[string]any{"applepieBlock": 0}}, nil)
	if err != nil {
		t.Fatalf("write applepie overlay: %v", err)
	}
	brioche, err := writeOverlay(ctx, dir, map[string]any{"config": map[string]any{"briocheBlock": 0}}, nil)
	if err != nil {
		t.Fatalf("write brioche overlay: %v", err)
	}
	if applepie == brioche {
		t.Fatalf("both overlays wrote to %s; the second overwrote the first", applepie)
	}
	for _, p := range []string{applepie, brioche} {
		if got := filepath.Dir(p); got != dir {
			t.Errorf("overlay written outside the data dir: %s", p)
		}
	}

	// The first file must still hold its own bytes after the second was written:
	// that is what makes the earlier environment's recorded request truthful.
	if again, err := writeOverlay(ctx, dir, map[string]any{"config": map[string]any{"applepieBlock": 0}}, nil); err != nil {
		t.Fatalf("rewrite applepie overlay: %v", err)
	} else if again != applepie {
		t.Errorf("the same overlay moved: %s then %s — the path must be a function of the content", applepie, again)
	}
}

// TestWriteOverlay_NoOverlayNamesNoFile keeps the empty case from writing a file
// whose path would then be digested as a genesis difference between two requests
// that both declare nothing.
func TestWriteOverlay_NoOverlayNamesNoFile(t *testing.T) {
	got, err := writeOverlay(context.Background(), t.TempDir(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("path = %q, want empty for a declaration with no overlay", got)
	}
}

// TestWriteOverlay_ProvidesIsWrittenAndIsPartOfTheIdentity carries the reason
// this parameter exists. A DSL env could shape the genesis but could not say
// what the shaping made the network able to do, so six cases declared the thing
// they needed as a REQUIREMENT the network never advertised and skipped for
// good. The list has to reach the file chainsetup reads, and two networks whose
// genesis is identical but whose advertised capabilities differ are not the
// same network — so it has to reach the path digest too.
func TestWriteOverlay_ProvidesIsWrittenAndIsPartOfTheIdentity(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	genesis := map[string]any{"config": map[string]any{"applepieBlock": 0}}

	bare, err := writeOverlay(ctx, dir, genesis, nil)
	if err != nil {
		t.Fatalf("write overlay without provides: %v", err)
	}
	withCap, err := writeOverlay(ctx, dir, genesis, []string{"short-expiry"})
	if err != nil {
		t.Fatalf("write overlay with provides: %v", err)
	}
	if bare == withCap {
		t.Fatal("the same path for two networks that advertise different capabilities")
	}

	var doc struct {
		Capabilities []string       `json:"capabilities"`
		Genesis      map[string]any `json:"genesis"`
	}
	b, err := os.ReadFile(withCap)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse overlay: %v", err)
	}
	if len(doc.Capabilities) != 1 || doc.Capabilities[0] != "short-expiry" {
		t.Errorf("capabilities = %v, want [short-expiry]", doc.Capabilities)
	}
	if doc.Genesis == nil {
		t.Error("the genesis half went missing when capabilities were added")
	}
}

// TestWriteOverlay_ProvidesAloneStillNamesAFile keeps an env that only
// advertises — it shapes nothing — from being dropped as empty, which would
// take its capability with it.
func TestWriteOverlay_ProvidesAloneStillNamesAFile(t *testing.T) {
	got, err := writeOverlay(context.Background(), t.TempDir(), nil, []string{"account-extra"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("no file for a declaration that advertises a capability")
	}
}
