package logs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/logs"
)

// writeLogs seeds <dir>/logs/node{1,2}.log with a few lines each.
func writeLogs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	node1 := "INFO [07-25|05:33:11.819] Starting Gstable\n" +
		"WARN [07-25|05:33:12.001] Deprecated personal namespace activated\n" +
		"    continued detail line with no level\n" +
		"ERROR [07-25|05:33:13.000] WBFT: invalid validator address=0xabc\n"
	node2 := "INFO [07-25|05:33:11.900] Started P2P networking\n" +
		"INFO [07-25|05:33:14.000] Commit new sealing work number=5\n"
	if err := os.WriteFile(filepath.Join(logDir, "node1.log"), []byte(node1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "node2.log"), []byte(node2), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSearch_All(t *testing.T) {
	dir := writeLogs(t)
	got, err := logs.Search(dir, logs.SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("want 6 lines, got %d", len(got))
	}
	// Ordered by node then line: first match is node1 line1.
	if got[0].Node != 1 || got[0].Line != 1 || got[0].Level != "INFO" {
		t.Errorf("first match: %+v", got[0])
	}
}

func TestSearch_LevelThreshold(t *testing.T) {
	dir := writeLogs(t)
	got, err := logs.Search(dir, logs.SearchOpts{Level: "WARN"})
	if err != nil {
		t.Fatal(err)
	}
	// WARN threshold keeps the WARN and ERROR lines, drops INFO and the
	// continuation line (no level).
	if len(got) != 2 {
		t.Fatalf("want 2 (WARN+ERROR), got %d: %+v", len(got), got)
	}
	for _, m := range got {
		if m.Level != "WARN" && m.Level != "ERROR" {
			t.Errorf("unexpected level in threshold result: %+v", m)
		}
	}
}

func TestSearch_NodeAndPattern(t *testing.T) {
	dir := writeLogs(t)
	got, err := logs.Search(dir, logs.SearchOpts{Node: 1, Pattern: "validator"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Node != 1 || got[0].Line != 4 {
		t.Fatalf("want node1 line4 only, got %+v", got)
	}
}

func TestSearch_Regexp(t *testing.T) {
	dir := writeLogs(t)
	got, err := logs.Search(dir, logs.SearchOpts{Pattern: `number=\d+`, Regexp: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Node != 2 {
		t.Fatalf("want one node2 match, got %+v", got)
	}
}

func TestSearch_Limit(t *testing.T) {
	dir := writeLogs(t)
	got, err := logs.Search(dir, logs.SearchOpts{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("limit not applied: %d", len(got))
	}
}

func TestSearch_NoLogsDir(t *testing.T) {
	got, err := logs.Search(t.TempDir(), logs.SearchOpts{})
	if err != nil {
		t.Fatalf("missing logs dir should not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want no matches, got %d", len(got))
	}
}

func TestSearch_BadPattern(t *testing.T) {
	dir := writeLogs(t)
	if _, err := logs.Search(dir, logs.SearchOpts{Pattern: "(", Regexp: true}); err == nil {
		t.Fatal("want error for invalid regexp")
	}
}
