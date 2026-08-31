package collector

import (
	"regexp"
	"sort"
)

// tsRE extracts the geth-family log timestamp "[MM-DD|HH:MM:SS.mmm]". The
// fixed-width form sorts chronologically as a plain string within a year.
var tsRE = regexp.MustCompile(`\[(\d\d-\d\d\|\d\d:\d\d:\d\d\.\d+)\]`)

// Timeline returns the matching lines across all nodes ordered chronologically
// by their log timestamp, then by node and line for lines sharing a timestamp or
// lacking one. This interleaves events from different nodes so a cross-node
// sequence (e.g. a consensus round, or a handoff) can be read in order.
//
// It searches the full set first and sorts before applying Limit, so the cap
// keeps the earliest lines rather than whichever node was scanned first.
func Timeline(dir string, opts SearchOpts) ([]SearchMatch, error) {
	full := opts
	full.Limit = 0
	matches, err := Search(dir, full)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(matches, func(i, j int) bool {
		ti, tj := lineTimestamp(matches[i].Text), lineTimestamp(matches[j].Text)
		if ti != tj {
			return ti < tj
		}
		if matches[i].Node != matches[j].Node {
			return matches[i].Node < matches[j].Node
		}
		return matches[i].Line < matches[j].Line
	})
	if opts.Limit > 0 && len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
	}
	return matches, nil
}

// lineTimestamp returns the "MM-DD|HH:MM:SS.mmm" timestamp of a log line, or ""
// when the line has none (sorts before any timestamped line).
func lineTimestamp(text string) string {
	if m := tsRE.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}
