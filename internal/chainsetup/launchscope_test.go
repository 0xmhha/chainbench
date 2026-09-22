package chainsetup_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
)

// A declared value has to reach the process as declared, or be refused. The
// third possibility — accepted, then quietly not what runs — is the one these
// cover.

// TestLaunchScope_RefusesAPerNodeKnobSharedBySeveralNodes is the defect this
// rule exists for.
//
// Measured before it: a two-node network with launch.all.port=39999 assembled
// BOTH nodes with --port 39999. The allocator had given them distinct slots and
// one override replaced both, so the second node could not bind and the network
// that came up was not the network declared. The same holds for a data root, a
// keystore, an account to unlock.
func TestLaunchScope_RefusesAPerNodeKnobSharedBySeveralNodes(t *testing.T) {
	for _, knob := range []string{"port=39999", "datadir=/tmp/one", "unlock=0x1"} {
		t.Run(knob, func(t *testing.T) {
			_, err := compose(t, map[string][]string{"all": {knob}})
			if err == nil {
				t.Fatalf("%q on scope all was accepted; it gives every node the same value", knob)
			}
			for _, want := range []string{"scope", "collide", "node1"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal should say %q, got: %v", want, err)
				}
			}
		})
	}
}

// TestLaunchScope_OneNodeMayBeToldItsOwn: naming a single node collapses
// nothing, and is a thing somebody may mean.
func TestLaunchScope_OneNodeMayBeToldItsOwn(t *testing.T) {
	args, err := compose(t, map[string][]string{"node1": {"port=39999"}})
	if err != nil {
		t.Fatalf("a per-node knob on one node was refused: %v", err)
	}
	if got, _ := flagValue(args[1], "--port"); got != "39999" {
		t.Errorf("node1 runs --port %s, and node1 was told 39999", got)
	}
	if got, _ := flagValue(args[2], "--port"); got == "39999" {
		t.Errorf("node2 also runs --port 39999, and only node1 was told")
	}
}

// TestLaunchScope_AKnobEveryNodeMayShareIsAllowed: the rule names the knobs one
// value cannot serve, not every knob the node's own facts set. Turning an
// endpoint on for the whole network is what a scope is for.
func TestLaunchScope_AKnobEveryNodeMayShareIsAllowed(t *testing.T) {
	args, err := compose(t, map[string][]string{"all": {"maxpeers=25"}})
	if err != nil {
		t.Fatalf("a shareable knob on scope all was refused: %v", err)
	}
	for i, a := range args {
		if got, _ := flagValue(a, "--maxpeers"); got != "25" {
			t.Errorf("node%d runs --maxpeers %q, and every node was told 25", i, got)
		}
	}
}

// compose stands a network up to deploy and returns each node's assembled argv,
// keyed by node index.
func compose(t *testing.T, scoped map[string][]string) (map[int][]string, error) {
	t.Helper()
	dir := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verb.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpDeploy,
		Chain: "stablenet", KeysDir: keysAbs, BPCount: 2, ENCount: 0,
		LaunchScoped: scoped,
	}); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
	if err != nil {
		t.Fatalf("read the record: %v", err)
	}
	var rec struct {
		Nodes []struct {
			Index int      `json:"index"`
			Args  []string `json:"args"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("parse the record: %v", err)
	}
	out := map[int][]string{}
	for _, n := range rec.Nodes {
		out[n.Index] = n.Args
	}
	return out, nil
}

// TestLaunchLadder_TheInvocationWinsOverAnyScope is the priority line, at the
// one place two orders met and disagreed.
//
// Scope narrowness says node beats role beats all; the line says the invocation
// beats every document. The invocation has no scope syntax — it speaks for all
// nodes — so a declaration that named a role used to beat it. Measured before
// this: a declaration with bp:maxpeers=11 and --launch-opt maxpeers=22 gave the
// producers 11, and the flag the operator typed did nothing on the nodes it was
// typed for.
func TestLaunchLadder_TheInvocationWinsOverAnyScope(t *testing.T) {
	for _, scope := range []string{"all", "bp", "node1"} {
		t.Run("declared on "+scope, func(t *testing.T) {
			args, err := composeWith(t, map[string][]string{scope: {"maxpeers=11"}}, []string{"maxpeers=22"})
			if err != nil {
				t.Fatalf("compose: %v", err)
			}
			for i, a := range args {
				got, ok := flagValue(a, "--maxpeers")
				if !ok {
					t.Fatalf("node%d was assembled without --maxpeers", i)
				}
				if got != "22" {
					t.Errorf("node%d runs --maxpeers %s; the declaration said 11 on scope %q and the invocation said 22", i, got, scope)
				}
			}
		})
	}
}

// TestLaunchLadder_ADeclarationStillWinsOverTheChain: the rung below still
// applies. With no invocation, what the declaration says is what runs.
func TestLaunchLadder_ADeclarationStillWinsOverTheChain(t *testing.T) {
	args, err := composeWith(t, map[string][]string{"all": {"maxpeers=11"}}, nil)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	for i, a := range args {
		if got, _ := flagValue(a, "--maxpeers"); got != "11" {
			t.Errorf("node%d runs --maxpeers %q, and the declaration said 11", i, got)
		}
	}
}

// composeWith stands a network up to deploy with both a declaration's scoped
// overrides and an invocation's, and returns each node's argv.
func composeWith(t *testing.T, scoped map[string][]string, set []string) (map[int][]string, error) {
	t.Helper()
	dir := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verb.NetUp(context.Background(), chainsetup.Deps{Clock: fixedClock()}, chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpDeploy,
		Chain: "stablenet", KeysDir: keysAbs, BPCount: 2, ENCount: 0,
		LaunchScoped: scoped, LaunchSet: set,
	}); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
	if err != nil {
		t.Fatalf("read the record: %v", err)
	}
	var rec struct {
		Nodes []struct {
			Index int      `json:"index"`
			Args  []string `json:"args"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("parse the record: %v", err)
	}
	out := map[int][]string{}
	for _, n := range rec.Nodes {
		out[n.Index] = n.Args
	}
	return out, nil
}
