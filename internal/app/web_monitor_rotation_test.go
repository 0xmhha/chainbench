package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// reusedInodeLog is a log whose replacement gets the deleted file's inode
// number, as ext4 routinely hands it out again.
type reusedInodeLog struct{ content string }

func (l *reusedInodeLog) Snapshot(_ context.Context, _ string, offset int64, limit int) (webLogSnapshot, error) {
	size := int64(len(l.content))
	s := webLogSnapshot{ID: 42, Size: size, Head: []byte(l.content[:min(size, webMonitorHeadBytes)])}
	if size > offset {
		s.Data = []byte(l.content[offset:min(offset+int64(limit), size)])
	}
	return s, nil
}

// A replacement larger than the old offset is read from its start even when
// the filesystem reused the inode number.
func TestWebMonitorLogRotationIsSeenWhenTheInodeIsReused(t *testing.T) {
	m := newMonitorFixture(t).monitor()
	cursor, err := m.readCursor("net")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	log := &reusedInodeLog{content: "INFO [10-10|00:00:00.000] old first\nINFO [10-10|00:00:01.000] old second\n"}
	if _, err = m.archiveLog(context.Background(), "net", "node1", "node1.log", log, &cursor, now, now, nil); err != nil {
		t.Fatal(err)
	}
	read := cursor.Offsets["node1"]
	log.content = strings.Repeat("INFO [10-10|00:00:05.000] new line\n", 10)
	if int64(len(log.content)) <= read {
		t.Fatal("the replacement must be larger than the old offset")
	}
	gaps, err := m.archiveLog(context.Background(), "net", "node1", "node1.log", log, &cursor, now, now.Add(time.Second), nil)
	if err != nil {
		t.Fatal(err)
	}
	reasons := []string{}
	for _, g := range gaps {
		reasons = append(reasons, g.Reason)
	}
	if !strings.Contains(strings.Join(reasons, ","), "log_rotated") {
		t.Fatalf("a replaced file with a reused inode was not reported: %v", reasons)
	}
	if cursor.Offsets["node1"] != int64(len(log.content)) {
		t.Fatalf("the replacement was read from byte %d instead of its start", read)
	}
}

// A log rewritten in place keeps its inode on every filesystem; the local and
// the remote reader both see that the bytes they read from are gone.
func TestWebMonitorLogRewrittenInPlaceIsReadFromItsStart(t *testing.T) {
	for name, source := range map[string]webLogSource{"local": webLocalLogs{}, "remote": webRemoteLogs{run: shellRunner}} {
		t.Run(name, func(t *testing.T) {
			m := newMonitorFixture(t).monitor()
			cursor, err := m.readCursor("net")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "node1.log")
			now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
			if err = os.WriteFile(path, []byte("INFO [10-10|00:00:00.000] old first\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err = m.archiveLog(context.Background(), "net", "node1", path, source, &cursor, now, now, nil); err != nil {
				t.Fatal(err)
			}
			rewritten := strings.Repeat("INFO [10-10|00:00:05.000] new line\n", 10)
			if err = os.WriteFile(path, []byte(rewritten), 0o600); err != nil {
				t.Fatal(err)
			}
			gaps, err := m.archiveLog(context.Background(), "net", "node1", path, source, &cursor, now, now.Add(time.Second), nil)
			if err != nil || len(gaps) != 1 || gaps[0].Reason != "log_rotated" || cursor.Offsets["node1"] != int64(len(rewritten)) {
				t.Fatalf("rewrite not read from its start: gaps=%v offset=%d err=%v", gaps, cursor.Offsets["node1"], err)
			}
			// Appending to the file it now reads is no rotation.
			if err = os.WriteFile(path, []byte(rewritten+"INFO [10-10|00:00:06.000] appended\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if gaps, err = m.archiveLog(context.Background(), "net", "node1", path, source, &cursor, now, now.Add(2*time.Second), nil); err != nil || len(gaps) != 0 {
				t.Fatalf("growth reported as a gap: %v %v", gaps, err)
			}
		})
	}
}
