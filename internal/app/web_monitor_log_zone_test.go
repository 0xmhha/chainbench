package app

import (
	"testing"
	"time"
)

// A node log stamp has no zone. It belongs to the node's machine, so a collector
// in another zone must not shift it.
func TestLogLine_ReadsAStampInTheServersZone(t *testing.T) {
	now := time.Date(2026, 11, 5, 8, 44, 0, 0, time.UTC)
	m := &WebMonitor{}
	line := m.logLine("INFO [11-05|08:43:04.041] start new round", now, webLogLocation("+0000"))
	want := time.Date(2026, 11, 5, 8, 43, 4, 41_000_000, time.UTC)
	if !line.Time.Equal(want) || line.TimestampSource != "source" {
		t.Fatalf("line = %+v, want %v", line, want)
	}
	line = m.logLine("INFO [11-05|17:43:04.041] start new round", now, webLogLocation("+0900"))
	if !line.Time.Equal(want) {
		t.Fatalf("line at +0900 = %v, want %v", line.Time, want)
	}
}

func TestWebLogLocation_FallsBackToTheLocalZone(t *testing.T) {
	for _, offset := range []string{"", "not-an-offset"} {
		if got := webLogLocation(offset); got != time.Local {
			t.Errorf("offset %q gave %v", offset, got)
		}
	}
}
