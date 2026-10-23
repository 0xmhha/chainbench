package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// LogLevel is the severity a geth-family log line starts with, normalized to
// one spelling. An empty level means the line carries none.
type LogLevel string

const (
	LogLevelCrit  LogLevel = "CRIT"
	LogLevelError LogLevel = "ERROR"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelTrace LogLevel = "TRACE"
)

// webLogSearchMaxLen bounds a search string; a longer one is refused rather
// than scanned against every archived line.
const webLogSearchMaxLen = 200

// geth 1.10 (go-wemix) writes the four-letter forms, geth 1.14 the full words;
// both pad to five columns and run straight into the bracketed timestamp.
var logLevelPrefix = regexp.MustCompile(`^(CRIT|ERROR|EROR|WARN|INFO|DEBUG|DBUG|TRACE|TRCE) *\[`)

var logLevelSpellings = map[string]LogLevel{
	"CRIT":  LogLevelCrit,
	"ERROR": LogLevelError, "EROR": LogLevelError,
	"WARN":  LogLevelWarn,
	"INFO":  LogLevelInfo,
	"DEBUG": LogLevelDebug, "DBUG": LogLevelDebug,
	"TRACE": LogLevelTrace, "TRCE": LogLevelTrace,
}

func parseLogLevel(line string) LogLevel {
	m := logLevelPrefix.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return logLevelSpellings[m[1]]
}

// ParseLogLevels reads a comma-separated list of level names in either case.
// An empty list means no level filter.
func ParseLogLevels(raw string) ([]LogLevel, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out []LogLevel
	for _, name := range strings.Split(raw, ",") {
		level, ok := logLevelSpellings[strings.ToUpper(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("unknown log level %q", strings.TrimSpace(name))
		}
		if !slices.Contains(out, level) {
			out = append(out, level)
		}
	}
	return out, nil
}

// webLogFilter selects archived lines by text and level. Matching runs on the
// redacted text, so a redacted secret is never found by searching for it.
type webLogFilter struct {
	needle string
	levels []LogLevel
}

func newWebLogFilter(q WebMonitorQuery) (webLogFilter, error) {
	if len(q.Search) > webLogSearchMaxLen {
		return webLogFilter{}, ErrWebMonitorQuery
	}
	levels := slices.Clone(q.Levels)
	slices.Sort(levels)
	return webLogFilter{needle: strings.ToLower(q.Search), levels: slices.Compact(levels)}, nil
}

func (f webLogFilter) match(text string, level LogLevel) bool {
	if len(f.levels) > 0 && !slices.Contains(f.levels, level) {
		return false
	}
	return f.needle == "" || strings.Contains(strings.ToLower(text), f.needle)
}

// fingerprint ties a cursor to the filter it was issued for, so a page is
// never continued under a different search.
func (f webLogFilter) fingerprint() string {
	if f.needle == "" && len(f.levels) == 0 {
		return ""
	}
	parts := make([]string, len(f.levels))
	for i, l := range f.levels {
		parts[i] = string(l)
	}
	sum := sha256.Sum256([]byte(f.needle + "\x00" + strings.Join(parts, ",")))
	return hex.EncodeToString(sum[:8])
}
