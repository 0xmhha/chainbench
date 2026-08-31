// Package logs is a search surface over the per-node log files a setup leaves
// under <data-dir>/logs/node<N>.log (requirement #5, replacing the legacy bash
// logs/*.sh). It parses the geth-family line format
// "LEVEL [MM-DD|HH:MM:SS.mmm] message  key=val" and filters by node, severity,
// and a substring/regexp pattern, so the CLI and MCP surfaces can answer
// "show me the errors on node 2" without shelling out.
package collector

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Match is one matching log line.
type Match struct {
	Node  int    `json:"node"`  // 1-based node index (from the file name)
	Line  int    `json:"line"`  // 1-based line number within that node's log
	Level string `json:"level"` // parsed severity (INFO/WARN/ERROR/...), "" if none
	Text  string `json:"text"`  // the full line, trailing newline stripped
}

// SearchOpts configures a Search.
type SearchOpts struct {
	Pattern string // substring (or regexp when Regexp) to match; "" matches all
	Regexp  bool   // treat Pattern as a regular expression
	Node    int    // restrict to this 1-based node; 0 = all nodes
	Level   string // minimum severity (e.g. "WARN"); "" = any level
	Limit   int    // cap results; <=0 = no cap
}

// levelRank orders the geth log severities so a Level filter means
// "this severity or higher".
var levelRank = map[string]int{
	"TRACE": 0, "DEBUG": 1, "INFO": 2, "WARN": 3, "ERROR": 4, "CRIT": 5,
}

// Search scans <dir>/logs/node*.log and returns the lines matching opts, in
// (node, line) order. A missing logs directory yields no matches (not an
// error): a setup that never launched simply has none.
func Search(dir string, opts SearchOpts) ([]Match, error) {
	logDir := filepath.Join(dir, "logs")
	files, err := filepath.Glob(filepath.Join(logDir, "node*.log"))
	if err != nil {
		return nil, fmt.Errorf("logs: glob: %w", err)
	}
	sort.Strings(files)

	var re *regexp.Regexp
	if opts.Pattern != "" && opts.Regexp {
		if re, err = regexp.Compile(opts.Pattern); err != nil {
			return nil, fmt.Errorf("logs: bad pattern: %w", err)
		}
	}

	minRank := -1
	if opts.Level != "" {
		r, ok := levelRank[strings.ToUpper(opts.Level)]
		if !ok {
			return nil, fmt.Errorf("logs: unknown level %q", opts.Level)
		}
		minRank = r
	}

	var out []Match
	for _, f := range files {
		node := nodeIndex(f)
		if opts.Node > 0 && node != opts.Node {
			continue
		}
		matches, err := scanFile(f, node, opts, re, minRank)
		if err != nil {
			return nil, err
		}
		out = append(out, matches...)
		if opts.Limit > 0 && len(out) >= opts.Limit {
			return out[:opts.Limit], nil
		}
	}
	return out, nil
}

func scanFile(path string, node int, opts SearchOpts, re *regexp.Regexp, minRank int) ([]Match, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("logs: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var out []Match
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // node logs can have long lines
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		level := parseLevel(text)
		if minRank >= 0 {
			// With a level filter, drop lines below the threshold and
			// continuation lines that carry no parseable level.
			if r, ok := levelRank[level]; !ok || r < minRank {
				continue
			}
		}
		if !matchesPattern(text, opts, re) {
			continue
		}
		out = append(out, Match{Node: node, Line: line, Level: level, Text: text})
		if opts.Limit > 0 && len(out) >= opts.Limit {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("logs: read %s: %w", path, err)
	}
	return out, nil
}

func matchesPattern(text string, opts SearchOpts, re *regexp.Regexp) bool {
	switch {
	case opts.Pattern == "":
		return true
	case re != nil:
		return re.MatchString(text)
	default:
		return strings.Contains(text, opts.Pattern)
	}
}

// parseLevel returns the leading severity token of a geth log line, or "" if
// the line does not start with a known level (e.g. a wrapped continuation).
func parseLevel(text string) string {
	tok := text
	if i := strings.IndexByte(text, ' '); i >= 0 {
		tok = text[:i]
	}
	tok = strings.ToUpper(tok)
	if _, ok := levelRank[tok]; ok {
		return tok
	}
	return ""
}

// nodeIndex extracts N from a ".../node<N>.log" path; 0 if unparsable.
func nodeIndex(path string) int {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".log")
	n, err := strconv.Atoi(strings.TrimPrefix(base, "node"))
	if err != nil {
		return 0
	}
	return n
}
