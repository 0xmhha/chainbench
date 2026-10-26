package app

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// sampleRun reads a run detail recorded from a real 4-node stablenet; see
// docs/dev/web-data-inventory.md for how the sample was taken.
func sampleRun(t *testing.T, name string) WebRun {
	t.Helper()
	raw, err := os.ReadFile("../../tests/webui/fixtures/stablenet-sample/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Body WebRun `json:"body"`
	}
	if err = decodeHistoryJSON(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Body
}

func TestWebRunTestsReadTheEngineResultFiles(t *testing.T) {
	failed := webRunTests(sampleRun(t, "history-session-dc79d156e026c1c3473a07396ec1825a"))
	want := []WebRunTest{{
		Seq: 1, ID: "sample-fails", State: "failed", Reason: "1 assertion(s) failed", Env: "env-b5f52a59ddb9",
		StartedAt: "2026-10-25T08:14:02.59831Z", FinishedAt: "2026-10-25T08:14:04.618204Z",
		Assertions: []WebAssertion{{Assert: "blockNumber", NodeID: "node1", Compare: "Less", Expected: "0", Actual: "7", Passed: false,
			Detail: "node1: actual=7 expected=0", Provenance: map[string]any{"assert": "blockNumber", "compare": "Less", "expected": json.Number("0"), "on": "node1"}}},
		Steps:       []WebStep{{Index: 0, Type: "read", NodeID: "node1"}},
		PostActions: []WebPostAction{},
		Artifacts:   []WebArtifactRef{{Kind: "genesis", Ref: "genesis.json"}},
	}}
	if !reflect.DeepEqual(failed, want) {
		t.Fatalf("tests = %#v\nwant    %#v", failed, want)
	}
	passed := webRunTests(sampleRun(t, "history-session-7cb268ea6436499fda20201a363d4627"))
	if len(passed) != 1 || passed[0].State != "succeeded" || passed[0].Assertions[0].Expected != "15" || !passed[0].Assertions[0].Passed {
		t.Fatalf("passing run tests = %#v", passed)
	}
}

func TestWebRunTestsKeepEveryValueAsDecimalText(t *testing.T) {
	run := WebRun{Summary: map[string]any{
		"tests": []any{map[string]any{"seq": 2, "id": "big", "status": "blocked"}},
		"results": map[string]any{
			"tests/002_big/assert.json": []any{map[string]any{"Assert": "balance", "On": "node2", "Expected": decodedNumber(t, "123456789012345678901234567890"), "Actual": "123456789012345678901234567891", "Pass": false}},
			"tests/002_big/steps.json":  []any{map[string]any{"Index": 0, "Type": "tx", "On": "node2", "Nonce": decodedNumber(t, "18446744073709551615"), "Gas": "21000", "Hash": "0xab", "Error": "reverted"}},
		},
	}}
	got := webRunTests(run)
	if len(got) != 1 || got[0].State != "blocked" {
		t.Fatalf("tests = %#v", got)
	}
	if a := got[0].Assertions[0]; a.Expected != "123456789012345678901234567890" || a.Actual != "123456789012345678901234567891" || a.NodeID != "node2" {
		t.Fatalf("assertion = %#v", a)
	}
	if s := got[0].Steps[0]; s.Nonce != "18446744073709551615" || s.Gas != "21000" || s.Hash != "0xab" || s.Error != "reverted" {
		t.Fatalf("step = %#v", s)
	}
}

func decodedNumber(t *testing.T, literal string) any {
	t.Helper()
	var v any
	if err := decodeHistoryJSON([]byte(literal), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestWebRunMissingDropsAnEndTheDetailFound(t *testing.T) {
	run := sampleRun(t, "history-session-dc79d156e026c1c3473a07396ec1825a")
	if got := webRunMissing(run); !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("missing = %q although finishedAt is %q", got, run.FinishedAt)
	}
	run.FinishedAt = ""
	if got := webRunMissing(run); !reflect.DeepEqual(got, []string{"finishedAt"}) {
		t.Fatalf("missing without an end = %q", got)
	}
}
