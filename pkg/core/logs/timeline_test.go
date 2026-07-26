package logs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/logs"
)

func writeLog(t *testing.T, dir string, node int, lines string) {
	t.Helper()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(logDir, "node"+string(rune('0'+node))+".log")
	if err := os.WriteFile(p, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTimeline_InterleavesByTimestamp(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, 1, "INFO [07-26|10:00:03.000] n1 third\nINFO [07-26|10:00:01.000] n1 first\n")
	writeLog(t, dir, 2, "INFO [07-26|10:00:02.000] n2 second\nINFO [07-26|10:00:04.000] n2 fourth\n")

	tl, err := logs.Timeline(dir, logs.SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tl) != 4 {
		t.Fatalf("want 4 lines, got %d", len(tl))
	}
	order := []string{"n1 first", "n2 second", "n1 third", "n2 fourth"}
	for i, want := range order {
		if !contains(tl[i].Text, want) {
			t.Errorf("position %d = %q, want %q", i, tl[i].Text, want)
		}
	}
}

func TestTimeline_LimitKeepsEarliest(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, 1, "INFO [07-26|10:00:05.000] late\n")
	writeLog(t, dir, 2, "INFO [07-26|10:00:01.000] early\n")
	tl, err := logs.Timeline(dir, logs.SearchOpts{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(tl) != 1 || !contains(tl[0].Text, "early") {
		t.Errorf("limit should keep the earliest line, got %+v", tl)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
