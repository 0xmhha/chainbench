package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// The person reading a node's log is looking for one line. The search runs over
// the whole archived window on the server, so "no match" means none in the
// window, not none in the page the browser happened to load.
func TestWebMonitorLogsSearchTheWholeWindowByTextAndLevel(t *testing.T) {
	f := newMonitorFixture(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	lines := strings.Join([]string{
		"INFO [10-09|00:46:20.597] Imported new chain segment number=1",
		"EROR [10-09|00:46:21.000] Failed to sync err=TimeOut",
		"WARN [10-09|00:46:22.000] Peer dropped reason=timeout key=secret-value",
		"ERROR[10-09|00:46:23.000] Engine stopped",
		"plain line without level",
	}, "\n") + "\n"
	if err := os.WriteFile(f.logPath, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	if err := m.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	node := f.network + ".node1"
	read := func(q WebMonitorQuery) WebLogs {
		t.Helper()
		logs, err := m.Logs(context.Background(), viewer, node, q)
		if err != nil {
			t.Fatalf("Logs(%+v): %v", q, err)
		}
		return logs
	}
	texts := func(l WebLogs) []string {
		out := make([]string, len(l.Entries))
		for i, e := range l.Entries {
			out[i] = e.Text
		}
		return out
	}

	all := read(WebMonitorQuery{})
	wantLevels := []LogLevel{LogLevelInfo, LogLevelError, LogLevelWarn, LogLevelError, ""}
	if len(all.Entries) != len(wantLevels) || all.Total != len(wantLevels) {
		t.Fatalf("unfiltered = %d entries, total %d", len(all.Entries), all.Total)
	}
	for i, e := range all.Entries {
		if e.Level != wantLevels[i] {
			t.Errorf("entry %d level = %q, want %q (%s)", i, e.Level, wantLevels[i], e.Text)
		}
	}

	if got := read(WebMonitorQuery{Search: "TIMEOUT"}); got.Total != 2 || len(got.Entries) != 2 {
		t.Fatalf("case-insensitive search = %v (total %d)", texts(got), got.Total)
	}
	if got := read(WebMonitorQuery{Levels: []LogLevel{LogLevelError}}); got.Total != 2 || got.Entries[0].Level != LogLevelError || got.Entries[1].Level != LogLevelError {
		t.Fatalf("EROR and ERROR are one level: %v", texts(got))
	}
	if got := read(WebMonitorQuery{Search: "timeout", Levels: []LogLevel{LogLevelWarn}}); got.Total != 1 || !strings.Contains(got.Entries[0].Text, "Peer dropped") {
		t.Fatalf("text and level together = %v", texts(got))
	}
	if got := read(WebMonitorQuery{Search: "secret-value"}); got.Total != 0 {
		t.Fatalf("a redacted secret is searchable: %v", texts(got))
	}

	first := read(WebMonitorQuery{Search: "timeout", Limit: 1})
	if len(first.Entries) != 1 || first.Total != 2 || first.NextCursor == nil {
		t.Fatalf("first page = %v, total %d, cursor %v", texts(first), first.Total, first.NextCursor)
	}
	second := read(WebMonitorQuery{Search: "timeout", Limit: 1, Cursor: *first.NextCursor})
	if len(second.Entries) != 1 || second.Entries[0].Text == first.Entries[0].Text || second.NextCursor != nil {
		t.Fatalf("second page = %v", texts(second))
	}
	if _, err := m.Logs(context.Background(), viewer, node, WebMonitorQuery{Search: "sync", Cursor: *first.NextCursor}); !errors.Is(err, ErrWebMonitorQuery) {
		t.Fatalf("a cursor reused with another search was accepted: %v", err)
	}
	if _, err := m.Logs(context.Background(), viewer, node, WebMonitorQuery{Search: strings.Repeat("x", webLogSearchMaxLen+1)}); !errors.Is(err, ErrWebMonitorQuery) {
		t.Fatalf("an oversized search was accepted: %v", err)
	}
}

func TestParseLogLevelReadsBothGethSpellings(t *testing.T) {
	for _, c := range []struct {
		line string
		want LogLevel
	}{
		{"CRIT [10-09|00:46:20.597] boom", LogLevelCrit},
		{"EROR [10-09|00:46:20.597] old", LogLevelError},
		{"ERROR[10-09|00:46:20.597] new", LogLevelError},
		{"WARN [10-09|00:46:20.597] w", LogLevelWarn},
		{"INFO [10-09|00:46:20.597] i", LogLevelInfo},
		{"DBUG [10-09|00:46:20.597] d", LogLevelDebug},
		{"DEBUG[10-09|00:46:20.597] d", LogLevelDebug},
		{"TRCE [10-09|00:46:20.597] t", LogLevelTrace},
		{"TRACE[10-09|00:46:20.597] t", LogLevelTrace},
		{"INFORMATION about nothing", ""},
		{"error: not a geth line", ""},
		{"", ""},
	} {
		if got := parseLogLevel(c.line); got != c.want {
			t.Errorf("parseLogLevel(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestParseLogLevelsRefusesUnknownNames(t *testing.T) {
	got, err := ParseLogLevels("error, warn")
	if err != nil || len(got) != 2 || got[0] != LogLevelError || got[1] != LogLevelWarn {
		t.Fatalf("ParseLogLevels = %v, %v", got, err)
	}
	if _, err = ParseLogLevels("fatal"); err == nil {
		t.Fatal("an unknown level was accepted")
	}
	if got, err = ParseLogLevels(""); err != nil || got != nil {
		t.Fatalf("empty = %v, %v", got, err)
	}
}
