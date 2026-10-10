package app

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func historyTests(run WebRun) []session.TestResult {
	b, _ := json.Marshal(run.Summary["tests"])
	var tests []session.TestResult
	_ = json.Unmarshal(b, &tests)
	return tests
}

// Compare reports compatible test verdicts only. Missing metric/log samples
// cannot become zero-valued differences or a successful performance comparison.
func (s *WebHistory) Compare(ids []string) (WebComparison, error) {
	out := WebComparison{RunIDs: append([]string{}, ids...), Comparable: true, Limitations: []string{"Metric and log comparisons are unavailable in these captures."}, Results: []map[string]any{}}
	if len(ids) < 2 || len(ids) > 4 {
		return out, errors.New("select two to four distinct runs")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runs, seen := []WebRun{}, map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return out, errors.New("select distinct runs")
		}
		seen[id] = true
		capture, found := s.state.Captures[id]
		if !found {
			return out, ErrDeploymentNotFound
		}
		runs = append(runs, capture.Run)
	}
	base := runs[0]
	for _, run := range runs {
		if !terminalWebJob(run.State) {
			out.Comparable = false
			out.Limitations = append(out.Limitations, run.ID+": a terminal result has not been recorded.")
		}
		if run.Chain == "" {
			out.Comparable = false
			out.Limitations = append(out.Limitations, run.ID+": chain identity was not recorded.")
		} else if run.Chain != base.Chain {
			out.Comparable = false
			out.Limitations = append(out.Limitations, run.ID+": chain differs from the first run.")
		}
		if run.Fingerprints["binary"] == "" || base.Fingerprints["binary"] == "" {
			out.Comparable = false
			out.Limitations = append(out.Limitations, run.ID+": binary fingerprint is missing.")
		} else if run.Fingerprints["binary"] != base.Fingerprints["binary"] {
			out.Comparable = false
			out.Limitations = append(out.Limitations, run.ID+": binary fingerprint differs.")
		}
		if !sameHistoryEnvironments(run, base) {
			out.Comparable = false
			out.Limitations = append(out.Limitations, run.ID+": environment fingerprints are missing or differ.")
		}
	}
	verdicts, duplicate := map[string]map[string]string{}, false
	for _, run := range runs {
		seenCase := map[string]bool{}
		for _, test := range historyTests(run) {
			if seenCase[test.ID] {
				duplicate = true
			}
			seenCase[test.ID] = true
			if verdicts[test.ID] == nil {
				verdicts[test.ID] = map[string]string{}
			}
			verdicts[test.ID][run.ID] = test.Status
		}
	}
	if duplicate {
		out.Comparable = false
		out.Limitations = append(out.Limitations, "Repeated case IDs cannot be aligned unambiguously.")
	}
	keys := []string{}
	for id := range verdicts {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	common := 0
	for _, id := range keys {
		complete := len(verdicts[id]) == len(runs)
		if complete {
			common++
		}
		out.Results = append(out.Results, map[string]any{"caseId": id, "statuses": verdicts[id], "complete": complete})
		if !complete {
			out.Limitations = append(out.Limitations, id+": case is absent from one or more runs.")
		}
	}
	if common == 0 {
		out.Comparable = false
		out.Limitations = append(out.Limitations, "No common test cases were recorded.")
	}
	return out, nil
}

func sameHistoryEnvironments(a, b WebRun) bool {
	values := func(run WebRun) []string {
		out := []string{}
		for key, value := range run.Fingerprints {
			if key != "binary" && key != "target" {
				out = append(out, value)
			}
		}
		sort.Strings(out)
		return out
	}
	x, y := values(a), values(b)
	if len(x) == 0 || len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
