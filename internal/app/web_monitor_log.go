package app

import (
	"bytes"
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
)

type webMonitorLogLine struct {
	Time            time.Time `json:"t"`
	CollectedAt     time.Time `json:"c"`
	TimestampSource string    `json:"ts"`
	Text            string    `json:"x"`
}

// archiveLog appends complete new lines of a local node log. A partial last
// line waits for its newline. A replaced or shrunk file is reported and read
// from its start; a line longer than one read is reported and skipped.
func (m *WebMonitor) archiveLog(network, label, path string, cursor *webMonitorCursor, since, now time.Time) ([]webMonitorGap, error) {
	var gaps []webMonitorGap
	gap := func(reason string) {
		gaps = append(gaps, webMonitorGap{From: since, To: now, Node: label, Source: "logs", Reason: reason})
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		gap("log_unavailable")
		return gaps, nil
	}
	if err != nil {
		return gaps, err
	}
	offset := cursor.Offsets[label]
	if id := webMonitorFileID(info); id != 0 {
		if previous, ok := cursor.Files[label]; ok && previous != id {
			gap("log_rotated")
			offset, cursor.Skipping[label], cursor.InSecret[label] = 0, false, false
		}
		cursor.Files[label] = id
	}
	if info.Size() < offset {
		gap("log_truncated")
		offset, cursor.Skipping[label] = 0, false
	}
	f, err := os.Open(path)
	if err != nil {
		return gaps, err
	}
	defer func() { _ = f.Close() }()
	chunk := make([]byte, webMonitorReadLimit)
	n, _ := f.ReadAt(chunk, offset)
	chunk = chunk[:n]
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
		if err = appendWebMonitorLines(m.store, network, day, byDay[day]); err != nil {
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
