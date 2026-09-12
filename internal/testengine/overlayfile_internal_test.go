package testengine

import (
	"context"
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

	applepie, err := writeOverlay(ctx, dir, map[string]any{"config": map[string]any{"applepieBlock": 0}})
	if err != nil {
		t.Fatalf("write applepie overlay: %v", err)
	}
	brioche, err := writeOverlay(ctx, dir, map[string]any{"config": map[string]any{"briocheBlock": 0}})
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
	if again, err := writeOverlay(ctx, dir, map[string]any{"config": map[string]any{"applepieBlock": 0}}); err != nil {
		t.Fatalf("rewrite applepie overlay: %v", err)
	} else if again != applepie {
		t.Errorf("the same overlay moved: %s then %s — the path must be a function of the content", applepie, again)
	}
}

// TestWriteOverlay_NoOverlayNamesNoFile keeps the empty case from writing a file
// whose path would then be digested as a genesis difference between two requests
// that both declare nothing.
func TestWriteOverlay_NoOverlayNamesNoFile(t *testing.T) {
	got, err := writeOverlay(context.Background(), t.TempDir(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("path = %q, want empty for a declaration with no overlay", got)
	}
}
