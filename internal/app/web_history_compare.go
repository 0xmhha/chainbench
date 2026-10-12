package app

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func historyTests(run WebRun) []session.TestResult {
	b, _ := json.Marshal(run.Summary["tests"])
	var tests []session.TestResult
	_ = json.Unmarshal(b, &tests)
	return tests
}

// Compare lines up compatible runs by case verdict, assertion, declaration,
// duration and archived node metrics. Missing samples or ends are reported as
// limitations, never as zero-valued differences. Log lines are not compared;
// each run's archived logs are read from its own detail.
func (s *WebHistory) Compare(a DeploymentActor, ids []string) (WebComparison, error) {
	out := WebComparison{RunIDs: append([]string{}, ids...), Comparable: true, Limitations: []string{}, Results: []map[string]any{}, Durations: map[string]*float64{}, Metrics: []WebMetricComparison{}}
	if len(ids) < 2 || len(ids) > 4 {
		return out, errors.New("select two to four distinct runs")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runs, seen, files := []WebRun{}, map[string]bool{}, map[string]map[string]string{}
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
		files[id] = capture.Files
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
		assertions := map[string]map[string]bool{}
		declarations := map[string]bool{}
		for _, run := range runs {
			spec, asserts := historyCaseFiles(files[run.ID], id)
			if spec != "" {
				declarations[manifestHash([]byte(spec))] = true
			}
			for _, a := range asserts {
				if assertions[a.ID] == nil {
					assertions[a.ID] = map[string]bool{}
				}
				assertions[a.ID][run.ID] = a.Pass
			}
		}
		if len(declarations) > 1 {
			out.Comparable = false
			out.Limitations = append(out.Limitations, id+": the case declaration differs between runs.")
		}
		out.Results = append(out.Results, map[string]any{"caseId": id, "statuses": verdicts[id], "complete": complete, "assertions": assertions})
		if !complete {
			out.Limitations = append(out.Limitations, id+": case is absent from one or more runs.")
		}
	}
	s.compareDurationsAndMetrics(a, runs, &out)
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

type historyAssertion struct {
	ID   string
	Pass bool
}

// historyCaseFiles reads one case's declaration and assertion results from a
// captured session, whose tests live under tests/<seq>_<case id>/.
func historyCaseFiles(files map[string]string, caseID string) (string, []historyAssertion) {
	var spec string
	var asserts []historyAssertion
	for name, text := range files {
		parts := strings.Split(name, "/")
		if len(parts) != 3 || parts[0] != "tests" {
			continue
		}
		_, dirCase, ok := strings.Cut(parts[1], "_")
		if !ok || dirCase != caseID {
			continue
		}
		switch parts[2] {
		case "spec.json":
			spec = text
		case "assert.json":
			_ = json.Unmarshal([]byte(text), &asserts)
		}
	}
	return spec, asserts
}

// compareDurationsAndMetrics adds each run's recorded length and the node
// metrics archived in its window. Different units for one metric cannot be
// set side by side, so they are named as a limitation.
func (s *WebHistory) compareDurationsAndMetrics(a DeploymentActor, runs []WebRun, out *WebComparison) {
	byName := map[string]*WebMetricComparison{}
	units := map[string]map[string]bool{}
	for _, run := range runs {
		w, ok := s.windowOf(run)
		if !ok || w.To.IsZero() {
			out.Durations[run.ID] = nil
			out.Limitations = append(out.Limitations, run.ID+": no recorded end, so its duration and metrics window are unknown.")
			continue
		}
		seconds := w.To.Sub(w.From).Seconds()
		out.Durations[run.ID] = &seconds
		if s.observations == nil {
			continue
		}
		metrics, err := s.observations.Metrics(a, w.Network, WebMonitorQuery{From: w.From, To: w.To})
		if err != nil {
			out.Limitations = append(out.Limitations, run.ID+": its archived metrics could not be read.")
			continue
		}
		sampled := false
		for _, series := range metrics.Series {
			key := series.Name + "\x00" + series.Source
			c := byName[key]
			if c == nil {
				c = &WebMetricComparison{Name: series.Name, Unit: series.Unit, Source: series.Source, Values: map[string]WebMetricSummary{}}
				byName[key] = c
			}
			if units[key] == nil {
				units[key] = map[string]bool{}
			}
			units[key][series.Unit] = true
			summary := c.Values[run.ID]
			for _, p := range series.Samples {
				if p.Value == nil {
					continue
				}
				v := *p.Value
				summary.Samples++
				sampled = true
				if summary.Min == nil || v < *summary.Min {
					summary.Min = &v
				}
				if summary.Max == nil || v > *summary.Max {
					summary.Max = &v
				}
				last := v
				summary.Last = &last
			}
			c.Values[run.ID] = summary
		}
		if !sampled {
			out.Limitations = append(out.Limitations, run.ID+": no node metrics were archived in its window.")
		}
	}
	keys := make([]string, 0, len(byName))
	for k := range byName {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if len(units[k]) > 1 {
			out.Limitations = append(out.Limitations, byName[k].Name+": recorded in different units, so it is not compared.")
			continue
		}
		out.Metrics = append(out.Metrics, *byName[k])
	}
}
