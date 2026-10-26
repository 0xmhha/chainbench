package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// WebRunTest is one test of an engine session as the Web model reads it: the
// session.json verdict with its result files already parsed, so a screen does
// not rebuild the engine's tests/<seq>_<id>/ layout or read PascalCase files.
type WebRunTest struct {
	Seq        int    `json:"seq"`
	ID         string `json:"id"`
	State      string `json:"state"`
	Reason     string `json:"reason,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Env        string `json:"env,omitempty"`
	// Assertions, Steps, PostActions and Artifacts are empty when the engine
	// wrote no such file for the test.
	Assertions  []WebAssertion   `json:"assertions"`
	Steps       []WebStep        `json:"steps"`
	PostActions []WebPostAction  `json:"postActions"`
	Artifacts   []WebArtifactRef `json:"artifacts"`
}

// WebAssertion is one assert.json entry. Expected and Actual are decimal text
// whatever JSON type the engine wrote, so a large chain quantity keeps every
// digit and the two sides compare as the same type.
type WebAssertion struct {
	Assert     string         `json:"assert"`
	NodeID     string         `json:"nodeId,omitempty"`
	Compare    string         `json:"compare,omitempty"`
	Expected   string         `json:"expected"`
	Actual     string         `json:"actual,omitempty"`
	Passed     bool           `json:"passed"`
	Detail     string         `json:"detail,omitempty"`
	Provenance map[string]any `json:"provenance,omitempty"`
}

// WebStep is one steps.json entry without the raw receipt, which stays in the
// run's export.
type WebStep struct {
	Index      int                         `json:"index"`
	Type       string                      `json:"type"`
	NodeID     string                      `json:"nodeId,omitempty"`
	Signer     string                      `json:"signer,omitempty"`
	Hash       string                      `json:"hash,omitempty"`
	Nonce      string                      `json:"nonce,omitempty"`
	Gas        string                      `json:"gas,omitempty"`
	Error      string                      `json:"error,omitempty"`
	Deployment *session.ContractDeployment `json:"deployment,omitempty"`
}

type WebPostAction struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type WebArtifactRef = session.ArtifactRef

// webTestStates maps session.json verdicts to the Web execution states.
var webTestStates = map[string]string{"pass": "succeeded", "fail": "failed", "blocked": "blocked", "skip": "skipped"}

// webRunTests reads the tests of an engine-session run from its captured
// summary; a job run has none.
func webRunTests(run WebRun) []WebRunTest {
	results, _ := run.Summary["results"].(map[string]any)
	out := []WebRunTest{}
	for _, t := range historyTests(run) {
		dir := fmt.Sprintf("tests/%03d_%s/", t.Seq, t.ID)
		state, ok := webTestStates[t.Status]
		if !ok {
			state = "unknown"
		}
		var status struct {
			Reason string `json:"reason"`
		}
		webResultFile(results, dir+"status.json", &status)
		test := WebRunTest{Seq: t.Seq, ID: t.ID, State: state, Reason: status.Reason, StartedAt: t.StartedAt, FinishedAt: t.FinishedAt, Env: t.Env,
			Assertions: []WebAssertion{}, Steps: []WebStep{}, PostActions: []WebPostAction{}, Artifacts: []WebArtifactRef{}}
		var asserts []struct {
			On, Assert, Source string
			Expected, Actual   any
			Pass               bool
			Provenance         map[string]any
		}
		webResultFile(results, dir+"assert.json", &asserts)
		for _, a := range asserts {
			node, _ := a.Provenance["on"].(string)
			if a.On != "" {
				node = a.On
			}
			compare, _ := a.Provenance["compare"].(string)
			test.Assertions = append(test.Assertions, WebAssertion{Assert: a.Assert, NodeID: node, Compare: compare, Expected: webDecimal(a.Expected),
				Actual: webDecimal(a.Actual), Passed: a.Pass, Detail: a.Source, Provenance: a.Provenance})
		}
		var steps []session.StepResult
		webResultFile(results, dir+"steps.json", &steps)
		for _, s := range steps {
			step := WebStep{Index: s.Index, Type: s.Type, NodeID: s.On, Signer: s.Signer, Hash: s.Hash, Gas: s.Gas, Error: s.Error, Deployment: s.Deployment}
			if s.Nonce != 0 {
				step.Nonce = fmt.Sprint(s.Nonce)
			}
			test.Steps = append(test.Steps, step)
		}
		var posts []session.PostResult
		webResultFile(results, dir+"postaction.json", &posts)
		for _, p := range posts {
			test.PostActions = append(test.PostActions, WebPostAction{Name: p.Name, OK: p.OK, Detail: p.Detail})
		}
		var artifacts struct {
			Refs []WebArtifactRef `json:"refs"`
		}
		webResultFile(results, dir+"artifacts.json", &artifacts)
		test.Artifacts = append(test.Artifacts, artifacts.Refs...)
		out = append(out, test)
	}
	return out
}

// webResultFile decodes one captured result file into out, keeping numbers
// as their literal text. A missing or malformed file leaves out unchanged.
func webResultFile(results map[string]any, name string, out any) {
	value, ok := results[name]
	if !ok {
		return
	}
	b, err := json.Marshal(value)
	if err == nil {
		_ = decodeHistoryJSON(b, out)
	}
}

// webDecimal writes a recorded value as text: a number keeps its literal
// digits, a string stays as it is, and anything else is its JSON.
func webDecimal(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// webRunMissing is the run's missing dimensions as of this read. The capture
// fixed them when the session was taken; an end the detail found since then
// is no longer missing.
func webRunMissing(run WebRun) []string {
	out := []string{}
	var list []string
	switch v := run.Summary["missingDimensions"].(type) {
	case []string:
		list = v
	case []any:
		for _, item := range v {
			name, _ := item.(string)
			list = append(list, name)
		}
	}
	for _, name := range list {
		if name == "" || (name == "finishedAt" && strings.TrimSpace(run.FinishedAt) != "") {
			continue
		}
		out = append(out, name)
	}
	return out
}
