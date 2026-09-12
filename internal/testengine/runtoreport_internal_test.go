package testengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chains, as a run does
)

// WA23 asked for a compose->run->report test that CI can run. The compose half is
// covered by the gate refusals in runsuite_gate_test.go, which are reachable
// without a chain. This is the other half: a spec taken to a verdict and a report,
// with no binary anywhere.
//
// The stub is the CHAIN, not the harness. That distinction is the whole reason
// this is worth having: a stubbed harness answers no RPC, so the readiness gate
// and every assertion pass or fail for reasons that have nothing to do with the
// code under test — a lifecycle test written that way earlier in this work passed
// while proving nothing. A real JSON-RPC server on a loopback port makes the
// interpreter, the assertions and the recorder do their actual jobs.

// chainStub answers the JSON-RPC methods a minimal spec reads. Each call is
// counted so a test can show the run really went through the wire.
type chainStub struct {
	mu    sync.Mutex
	calls map[string]int
	head  string
}

func newChainStub(head string) *chainStub {
	return &chainStub{calls: map[string]int{}, head: head}
}

func (s *chainStub) count(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[method]
}

func (s *chainStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     any    `json:"id"`
		Method string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.calls[req.Method]++
	s.mu.Unlock()

	var result any
	switch req.Method {
	case "eth_blockNumber":
		result = s.head
	case "eth_chainId":
		result = "0x205b" // 8283, the stablenet id
	case "net_peerCount":
		result = "0x3"
	case "net_version":
		result = "8283"
	case "eth_syncing":
		result = false
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": req.ID,
			"error": map[string]any{"code": -32601, "message": "the stub does not answer " + req.Method},
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
}

// attachSpec is a v2 case that needs nothing but a reachable endpoint.
func attachSpec(t *testing.T, id string, expected string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "case", "id": id,
		"requires": []string{"rpc"},
		"env": map[string]any{
			"schemaVersion": "2", "kind": "env", "id": "e", "chain": "stablenet",
		},
		"steps": []map[string]any{
			{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": expected},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// runAgainstStub runs one spec through the attach engine against the stub and
// returns the session root.
func runAgainstStub(t *testing.T, stub *chainStub, spec []byte) (string, error) {
	t.Helper()
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)

	eng, err := NewAttachEngine(AttachConfig{
		Chain:        "stablenet",
		RPCURLs:      []string{srv.URL},
		ArtifactRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("attach engine: %v", err)
	}
	return eng.Run(context.Background(), [][]byte{spec})
}

// readJSON decodes a file the run recorded.
func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // a path this test just created
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Base(path), err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s does not parse: %v", filepath.Base(path), err)
	}
}

// TestRunToReport_APassingSpecIsRecordedWithItsEvidence takes a spec all the way
// to the artifacts a reader would open, which is the part no non-live test
// reached: a verdict in session.json and the assertion's own numbers beside it.
func TestRunToReport_APassingSpecIsRecordedWithItsEvidence(t *testing.T) {
	stub := newChainStub("0x9") // head 9
	root, err := runAgainstStub(t, stub, attachSpec(t, "head-is-past-five", "5"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stub.count("eth_blockNumber") == 0 {
		t.Error("the run never asked the chain for a block number — the assertion did not reach the wire")
	}

	var sess struct {
		Tests []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tests"`
		Summary struct {
			Pass int `json:"pass"`
			Fail int `json:"fail"`
		} `json:"summary"`
	}
	readJSON(t, filepath.Join(root, "session.json"), &sess)
	if sess.Summary.Pass != 1 || sess.Summary.Fail != 0 {
		t.Fatalf("summary = %+v, want one pass", sess.Summary)
	}
	if len(sess.Tests) != 1 || sess.Tests[0].Status != "pass" {
		t.Fatalf("tests = %+v, want one passing entry", sess.Tests)
	}

	// The evidence, not just the verdict: a report that says "pass" without the
	// numbers behind it is what made a forked network read as healthy.
	dir := testDir(t, root, "head-is-past-five")
	var asserts []struct {
		Assert string `json:"Assert"`
		Actual any    `json:"Actual"`
		Pass   bool   `json:"Pass"`
	}
	readJSON(t, filepath.Join(dir, "assert.json"), &asserts)
	if len(asserts) != 1 {
		t.Fatalf("recorded %d assertions, want 1", len(asserts))
	}
	if !asserts[0].Pass || asserts[0].Assert != "blockNumber" {
		t.Errorf("assertion record = %+v", asserts[0])
	}
	if fmt.Sprint(asserts[0].Actual) == "" {
		t.Error("the recorded assertion carries no actual value")
	}
}

// TestRunToReport_AFailingSpecRecordsWhy is the half that matters more. A run that
// reports "fail" and no reason sends the reader to the logs of something that did
// not happen, which is what WA12 was about; this shows the reason and the numbers
// survive all the way into the files.
func TestRunToReport_AFailingSpecRecordsWhy(t *testing.T) {
	stub := newChainStub("0x1") // head 1, so "at least 5" cannot hold
	// Run returns no error: a failed test is a RESULT, not an engine failure.
	// The engine records and the surface decides — the CLI turns a failed summary
	// into a non-zero exit and MCP into a tool error (WA7). Asserting an error
	// here would pin the wrong contract, and a reader who believed it would look
	// for the verdict in the wrong place.
	root, err := runAgainstStub(t, stub, attachSpec(t, "head-is-past-five", "5"))
	if err != nil {
		t.Fatalf("a failing spec is a result, not an engine error: %v", err)
	}

	var sess struct {
		Summary struct {
			Pass int `json:"pass"`
			Fail int `json:"fail"`
		} `json:"summary"`
	}
	readJSON(t, filepath.Join(root, "session.json"), &sess)
	if sess.Summary.Fail != 1 || sess.Summary.Pass != 0 {
		t.Fatalf("summary = %+v, want one failure", sess.Summary)
	}

	dir := testDir(t, root, "head-is-past-five")
	var status struct {
		Result string `json:"result"`
		Reason string `json:"reason"`
	}
	readJSON(t, filepath.Join(dir, "status.json"), &status)
	if status.Result != "fail" {
		t.Fatalf("result = %q, want fail", status.Result)
	}
	if !strings.Contains(status.Reason, "assertion") {
		t.Errorf("reason = %q, want it to say an assertion failed", status.Reason)
	}

	var asserts []struct {
		Expected any  `json:"Expected"`
		Actual   any  `json:"Actual"`
		Pass     bool `json:"Pass"`
	}
	readJSON(t, filepath.Join(dir, "assert.json"), &asserts)
	if len(asserts) != 1 || asserts[0].Pass {
		t.Fatalf("assertion record = %+v, want one failing entry", asserts)
	}
	if fmt.Sprint(asserts[0].Expected) != "5" {
		t.Errorf("expected value not recorded: %+v", asserts[0])
	}
}

// TestRunToReport_AnUnansweredMethodFailsTheSpecRatherThanTheHarness pins what
// happens when the chain cannot answer. The stub refuses anything outside its
// vocabulary, and the run has to turn that into a recorded verdict — not a panic
// and not a silent pass.
func TestRunToReport_AnUnansweredMethodFailsTheSpecRatherThanTheHarness(t *testing.T) {
	stub := newChainStub("0x9")
	spec, err := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "case", "id": "asks-for-what-the-stub-refuses",
		"requires": []string{"rpc"},
		"env": map[string]any{
			"schemaVersion": "2", "kind": "env", "id": "e", "chain": "stablenet",
		},
		"steps": []map[string]any{
			{"expect": "rpcCall", "method": "eth_getBalance", "params": []any{"0x0", "latest"}, "compare": "Equal", "is": "0x0"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	root, runErr := runAgainstStub(t, stub, spec)
	if runErr != nil {
		t.Fatalf("a refused call is a test result, not an engine error: %v", runErr)
	}
	dir := testDir(t, root, "asks-for-what-the-stub-refuses")
	var status struct {
		Result string `json:"result"`
		Reason string `json:"reason"`
	}
	readJSON(t, filepath.Join(dir, "status.json"), &status)
	if status.Result == "pass" {
		t.Fatalf("result = pass; a refused call must not read as a passing assertion")
	}
	if status.Reason == "" {
		t.Error("the failure carries no reason")
	}
}

// testDir finds the record directory a session wrote for one spec id.
func testDir(t *testing.T, root, id string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "tests"))
	if err != nil {
		t.Fatalf("read tests dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), id) {
			return filepath.Join(root, "tests", e.Name())
		}
	}
	t.Fatalf("no record directory for %q in %v", id, entries)
	return ""
}

// TestRunToReport_AFailedTestIsAResultNotAnEngineError pins the boundary the two
// tests above depend on. Run's error means "could not run"; a test that ran and
// failed is recorded, and the summary is what a surface reads to choose an exit
// code. Collapsing the two would make a failing suite indistinguishable from a
// broken harness.
func TestRunToReport_AFailedTestIsAResultNotAnEngineError(t *testing.T) {
	stub := newChainStub("0x1")
	root, err := runAgainstStub(t, stub, attachSpec(t, "cannot-hold", "99"))
	if err != nil {
		t.Fatalf("Run returned an error for a failed test: %v", err)
	}
	var sess struct {
		Summary struct {
			Pass int `json:"pass"`
			Fail int `json:"fail"`
		} `json:"summary"`
	}
	readJSON(t, filepath.Join(root, "session.json"), &sess)
	if sess.Summary.Fail != 1 {
		t.Fatalf("summary = %+v, want the failure recorded", sess.Summary)
	}
}
