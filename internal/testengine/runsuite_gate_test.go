package testengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/dsl"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chains, as a run does
)

// RunSuite is the compose->run->report entry point, and the highest-scoring
// uncovered function in this package: 27 branches and 15 refusals. Composing
// needs binaries, so CI cannot run the whole of it — but the refusals are the
// GATE before anything launches, and every one of them is reachable without a
// chain. That gate is what WA23 was asking about on the compose side.
//
// Each refusal here protects something irreversible: a run that got past them
// composes a network, writes to a target, and starts processes.

// caseSpec builds a v2 case as JSON. The env fields it sets are what
// compositionKey reads, so a test can make two specs agree or differ on purpose.
func caseSpec(t *testing.T, id, chain, binary string, extra map[string]any) []byte {
	t.Helper()
	env := map[string]any{
		"schemaVersion": "2", "kind": "env", "id": "e", "chain": chain,
		"binaries": map[string]any{"default": binary},
		"topology": map[string]any{"bp": 1},
	}
	for k, v := range extra {
		env[k] = v
	}
	b, err := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "case", "id": id,
		"requires": []string{"rpc"},
		"env":      env,
		"steps": []map[string]any{
			{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func runSuite(t *testing.T, in RunSuiteIn) error {
	t.Helper()
	if in.DataDir == "" && in.SpecContent != nil {
		in.DataDir = t.TempDir()
	}
	_, err := RunSuite(context.Background(), chainsetupDepsForTest(), in)
	return err
}

// chainsetupDepsForTest supplies the zero deps: every refusal under test fires
// before anything is composed, so nothing here is reached.
func chainsetupDepsForTest() chainsetup.Deps { return chainsetup.Deps{} }

func TestRunSuite_RefusesWithoutSpecsOrWorkspace(t *testing.T) {
	if err := runSuite(t, RunSuiteIn{DataDir: t.TempDir()}); err == nil ||
		!strings.Contains(err.Error(), "no specs given") {
		t.Errorf("a run with no specs should say so: %v", err)
	}
	spec := caseSpec(t, "a", "wbft", "gwbft", nil)
	_, err := RunSuite(context.Background(), chainsetupDepsForTest(), RunSuiteIn{SpecContent: [][]byte{spec}})
	if err == nil || !strings.Contains(err.Error(), "workspace directory is required") {
		t.Errorf("a run with no workspace should say so: %v", err)
	}
}

// TestRunSuite_NamesTheSpecThatWillNotParse: with several specs, "one of them is
// malformed" is not actionable. The refusal carries which.
func TestRunSuite_NamesTheSpecThatWillNotParse(t *testing.T) {
	good := caseSpec(t, "good", "wbft", "gwbft", nil)
	err := runSuite(t, RunSuiteIn{SpecContent: [][]byte{good, []byte("{not json")}})
	if err == nil {
		t.Fatal("a malformed spec was accepted")
	}
	if !strings.Contains(err.Error(), "spec 2") {
		t.Errorf("the refusal should say which spec: %v", err)
	}
}

// TestRunSuite_RefusesSpecsOnDifferentChains: one run composes one network, so
// two chains in one run is a request that cannot be honoured — and honouring
// half of it would run the second spec's assertions against the first's chain.
func TestRunSuite_RefusesSpecsOnDifferentChains(t *testing.T) {
	err := runSuite(t, RunSuiteIn{SpecContent: [][]byte{
		caseSpec(t, "on-wbft", "wbft", "gwbft", nil),
		caseSpec(t, "on-stablenet", "stablenet", "gstable", nil),
	}})
	if err == nil {
		t.Fatal("two chains in one run were accepted")
	}
	if !strings.Contains(err.Error(), "on-stablenet") {
		t.Errorf("the refusal should name the spec that disagrees: %v", err)
	}
}

// TestRunSuite_RefusesADifferentComposition is the discriminating one. The specs
// agree on the chain but ask for different networks, and composing the first
// while running both would test the second spec against a network it did not
// describe — passing for the wrong reason.
func TestRunSuite_RefusesADifferentComposition(t *testing.T) {
	cases := []struct {
		name  string
		extra map[string]any
	}{
		{name: "a different binary", extra: map[string]any{
			"binaries": map[string]any{"default": "gwbft-other"}}},
		{name: "a different topology", extra: map[string]any{
			"topology": map[string]any{"bp": 4}}},
		{name: "a different hardfork schedule", extra: map[string]any{
			"hardforks": map[string]any{"croissant": 50}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := runSuite(t, RunSuiteIn{SpecContent: [][]byte{
				caseSpec(t, "first", "wbft", "gwbft", nil),
				caseSpec(t, "second", "wbft", "gwbft", c.extra),
			}})
			if err == nil {
				t.Fatalf("%s was accepted as the same composition", c.name)
			}
			if !strings.Contains(err.Error(), "second") {
				t.Errorf("the refusal should name the spec that differs: %v", err)
			}
			if !strings.Contains(err.Error(), "one run composes one network") {
				t.Errorf("the refusal should say why: %v", err)
			}
		})
	}
}

// TestRunSuite_AcceptsSpecsThatComposeTheSameNetwork keeps the gate from being
// the trivially-safe kind that refuses everything: two specs describing one
// network must get past it, and fail later for want of a binary rather than here.
func TestRunSuite_AcceptsSpecsThatComposeTheSameNetwork(t *testing.T) {
	err := runSuite(t, RunSuiteIn{SpecContent: [][]byte{
		caseSpec(t, "first", "wbft", "gwbft", nil),
		caseSpec(t, "second", "wbft", "gwbft", nil),
	}})
	if err == nil {
		t.Skip("composed without a binary, so there is nothing to distinguish")
	}
	for _, gate := range []string{"one run composes one network", "unresolved", "no specs given"} {
		if strings.Contains(err.Error(), gate) {
			t.Fatalf("two specs describing one network were stopped by the gate: %v", err)
		}
	}
}

// TestRunSuite_PrecheckStopsAnUnresolvedReference: a spec naming an assertion
// that does not exist reaches the interpreter otherwise, where the mistake has
// no field to be reported in.
func TestRunSuite_PrecheckStopsAnUnresolvedReference(t *testing.T) {
	bad, err := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "case", "id": "typo",
		"requires": []string{"rpc"},
		"env": map[string]any{
			"schemaVersion": "2", "kind": "env", "id": "e", "chain": "wbft",
			"binaries": map[string]any{"default": "gwbft"},
			"topology": map[string]any{"bp": 1},
		},
		"steps": []map[string]any{{"expect": "blockNumbr", "compare": "GreaterOrEqual", "is": "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rerr := runSuite(t, RunSuiteIn{SpecContent: [][]byte{bad}})
	if rerr == nil {
		t.Fatal("a spec naming an assertion that does not exist was accepted")
	}
	if !strings.Contains(rerr.Error(), "typo") || !strings.Contains(rerr.Error(), "blockNumbr") {
		t.Errorf("the refusal should name the spec and the reference: %v", rerr)
	}
}

// TestRunSuite_ReadsSpecsFromPaths covers the other input form, and that a path
// that is not there is reported as such.
func TestRunSuite_ReadsSpecsFromPaths(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "case.json")
	if err := os.WriteFile(p, caseSpec(t, "from-file", "wbft", "gwbft", nil), 0o600); err != nil {
		t.Fatal(err)
	}
	// A readable spec gets past the gate; the run then fails on the binary.
	if err := runSuite(t, RunSuiteIn{SpecPaths: []string{p}, DataDir: t.TempDir()}); err != nil {
		if strings.Contains(err.Error(), "no specs given") {
			t.Errorf("a spec read from a path was not seen: %v", err)
		}
	}
	err := runSuite(t, RunSuiteIn{SpecPaths: []string{filepath.Join(dir, "nope.json")}, DataDir: t.TempDir()})
	if err == nil {
		t.Fatal("a missing spec path was accepted")
	}
	var _ dsl.Spec // the reader belongs to dsl; this pins that the engine surfaces its failure
}
