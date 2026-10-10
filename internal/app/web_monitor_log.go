package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

// webMonitorReadLimit bounds one log read so a large backlog is archived over
// several ticks; webMonitorLineLimit bounds one stored line.
const (
	webMonitorReadLimit = 1 << 20
	webMonitorLineLimit = 64 << 10
	// webMonitorHeadBytes of a log's start identify it together with its inode.
	webMonitorHeadBytes = 256
)

// webPendingLines are archived log lines waiting for the monitor lock.
type webPendingLines struct {
	record string
	lines  []webMonitorLogLine
}

type webMonitorLogLine struct {
	Time            time.Time `json:"t"`
	CollectedAt     time.Time `json:"c"`
	TimestampSource string    `json:"ts"`
	Text            string    `json:"x"`
}

// webLogSource reads a node log file where it lives: the local filesystem or
// a remote host through the collecting job's own SSH access.
type webLogSource interface {
	// Snapshot returns the file identity, size and first bytes together with
	// at most limit bytes from offset, observed in one step; os.ErrNotExist
	// when absent.
	Snapshot(ctx context.Context, path string, offset int64, limit int) (webLogSnapshot, error)
}

// webLogSnapshot is one observation of a node log.
type webLogSnapshot struct {
	ID   uint64
	Size int64
	// Head is the file's first webMonitorHeadBytes, fewer while it is shorter.
	// A filesystem hands a deleted file's inode number to the next file, so
	// the head is what tells a replacement from the file being read.
	Head []byte
	Data []byte
}

type webLocalLogs struct{}

func (webLocalLogs) Snapshot(_ context.Context, path string, offset int64, limit int) (webLogSnapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return webLogSnapshot{}, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return webLogSnapshot{}, err
	}
	s := webLogSnapshot{ID: webMonitorFileID(info), Size: info.Size()}
	head := make([]byte, min(info.Size(), webMonitorHeadBytes))
	n, err := f.ReadAt(head, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return webLogSnapshot{}, err
	}
	s.Head = head[:n]
	if info.Size() <= offset {
		return s, nil
	}
	chunk := make([]byte, limit)
	n, err = f.ReadAt(chunk, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return webLogSnapshot{}, err
	}
	s.Data = chunk[:n]
	return s, nil
}

// sameWebLogHead reports whether two heads of one growing file agree; the
// shorter was taken while the file was shorter.
func sameWebLogHead(a, b string) bool {
	n := min(len(a), len(b))
	return a[:n] == b[:n]
}

// archiveLog appends complete new lines of a node log. A partial last line
// waits for its newline. A replaced or shrunk file is reported and read from
// its start; a line longer than one read is reported and skipped.
// With pending set, new lines are buffered for the caller to append under the
// monitor lock; otherwise the caller already holds it and they are appended now.
func (m *WebMonitor) archiveLog(ctx context.Context, network, label, path string, source webLogSource, cursor *webMonitorCursor, since, now time.Time, pending *[]webPendingLines) ([]webMonitorGap, error) {
	var gaps []webMonitorGap
	gap := func(reason string) {
		gaps = append(gaps, webMonitorGap{From: since, To: now, Node: label, Source: "logs", Reason: reason})
	}
	offset := cursor.Offsets[label]
	snap, err := source.Snapshot(ctx, path, offset, webMonitorReadLimit)
	id, size, chunk := snap.ID, snap.Size, snap.Data
	if errors.Is(err, os.ErrNotExist) {
		gap("log_unavailable")
		return gaps, nil
	}
	if err != nil {
		return gaps, err
	}
	restart := false
	if id != 0 {
		if previous, ok := cursor.Files[label]; ok && previous != id {
			gap("log_rotated")
			restart, cursor.InSecret[label] = true, false
		}
		cursor.Files[label] = id
	}
	head := hex.EncodeToString(snap.Head)
	if previous := cursor.Heads[label]; !restart && size >= offset && previous != "" && !sameWebLogHead(previous, head) {
		gap("log_rotated")
		restart, cursor.InSecret[label] = true, false
	}
	cursor.Heads[label] = head
	if size < offset {
		gap("log_truncated")
		restart = true
	}
	if restart {
		cursor.Skipping[label] = false
	}
	if restart && offset != 0 {
		offset = 0
		if snap, err = source.Snapshot(ctx, path, 0, webMonitorReadLimit); err != nil {
			return gaps, err
		}
		size, chunk, cursor.Heads[label] = snap.Size, snap.Data, hex.EncodeToString(snap.Head)
	}
	if size <= offset {
		cursor.Offsets[label] = offset
		return gaps, nil
	}
	n := len(chunk)
	if cursor.Skipping[label] {
		cut := bytes.IndexByte(chunk, '\n')
		if cut < 0 {
			cursor.Offsets[label] = offset + int64(n)
			return gaps, nil
		}
		offset += int64(cut + 1)
		chunk = chunk[cut+1:]
		cursor.Skipping[label] = false
	}
	end := bytes.LastIndexByte(chunk, '\n')
	if end < 0 {
		if n == webMonitorReadLimit {
			gap("log_line_oversized")
			cursor.Skipping[label] = true
			offset += int64(len(chunk))
		}
		cursor.Offsets[label] = offset
		return gaps, nil
	}
	byDay := map[string][]webMonitorLogLine{}
	var days []string
	for _, text := range strings.Split(string(chunk[:end]), "\n") {
		line := m.logLine(m.redactLogLine(label, text, cursor), now)
		day := webMonitorDay("logs/"+label, line.Time) + ".jsonl"
		if _, ok := byDay[day]; !ok {
			days = append(days, day)
		}
		byDay[day] = append(byDay[day], line)
	}
	for _, day := range days {
		if pending != nil {
			*pending = append(*pending, webPendingLines{record: day, lines: byDay[day]})
		} else if err = appendWebMonitorLines(m.store, network, day, byDay[day]); err != nil {
			return gaps, err
		}
	}
	cursor.Offsets[label] = offset + int64(end+1)
	return gaps, nil
}

// redactLogLine applies the shared redaction and hides every line of a PEM
// block, whose body spans lines no single-line pattern can recognise.
func (m *WebMonitor) redactLogLine(label, text string, cursor *webMonitorCursor) string {
	if cursor.InSecret[label] {
		if strings.Contains(text, "-----END ") {
			cursor.InSecret[label] = false
		}
		return "[redacted PEM block]"
	}
	if i := strings.Index(text, "-----BEGIN "); i >= 0 && !strings.Contains(text[i:], "-----END ") {
		cursor.InSecret[label] = true
		text = text[:i] + "[redacted PEM block]"
	}
	text = m.redact(text)
	if len(text) > webMonitorLineLimit {
		text = text[:webMonitorLineLimit] + " [truncated " + strconv.Itoa(len(text)-webMonitorLineLimit) + " bytes]"
	}
	return text
}

// logLine keeps the geth-family source timestamp when present. Those stamps
// carry no year or zone: they are read in the collector's local zone with the
// collection year, moved back a year when that would lie in the future.
func (m *WebMonitor) logLine(text string, now time.Time) webMonitorLogLine {
	line := webMonitorLogLine{Time: now, CollectedAt: now, TimestampSource: "collection", Text: text}
	stamp := collector.LineTimestamp(text)
	if stamp == "" {
		return line
	}
	local := now.In(time.Local)
	parsed, err := time.ParseInLocation("2006-01-02|15:04:05.999999999", strconv.Itoa(local.Year())+"-"+stamp, time.Local)
	if err != nil {
		return line
	}
	if parsed.After(local.Add(24 * time.Hour)) {
		parsed = parsed.AddDate(-1, 0, 0)
	}
	line.Time, line.TimestampSource = parsed.UTC(), "source"
	return line
}

// webMonitorFileID identifies the log file itself, so a replacement that is
// already larger than the old offset is not read from its middle.
func webMonitorFileID(info os.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Ino)
	}
	return 0
}
