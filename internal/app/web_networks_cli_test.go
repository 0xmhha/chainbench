package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// statusCLI answers `chain status` with a fixed state per workspace directory,
// standing in for the CLI the dashboard runs.
type statusCLI map[string]State

func (c statusCLI) ChainStatus(_ context.Context, dir string) (State, error) { return c[dir], nil }

// networksFixture is an engine over one owned workspace whose control directory
// holds a record naming recordPID for node1.
func networksFixture(t *testing.T, recordPID int) (*WebChainEngine, string) {
	t.Helper()
	root := t.TempDir()
	s, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	set, err := s.SaveDocument(a, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	config, err := s.SaveDocument(a, "", 0, deploymentTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.SaveWorkspace(a, "", 0, DeploymentWorkspaceInput{Name: "Owned", Documents: []DeploymentDocumentRef{{set.ID, set.Revision}, {config.ID, config.Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "networks", w.ID)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	record, _ := json.Marshal(State{Chain: "stablenet", Nodes: []node.Record{{Index: 1, Label: "node1", Role: "en", PID: recordPID}}})
	if err = os.WriteFile(filepath.Join(dir, "chain-record.json"), record, 0o600); err != nil {
		t.Fatal(err)
	}
	return NewWebChainEngine(root, "", s, nil, nil, nil), dir
}

// TestWebNetworksShowWhatTheCLIReports: the network list shows the state
// `chainbench chain status` reports, so the Web and a terminal agree on a
// node's pid even when the record on disk is behind the process ledger.
func TestWebNetworksShowWhatTheCLIReports(t *testing.T) {
	engine, dir := networksFixture(t, 111)
	engine.UseCLI(statusCLI{dir: {Chain: "stablenet", Nodes: []node.Record{{Index: 1, Label: "node1", Role: "en", PID: 222}}}})
	networks, err := engine.Networks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 1 || len(networks[0].Nodes) != 1 || networks[0].Nodes[0].PID != 222 {
		t.Fatalf("networks = %+v, want node1 with the pid the CLI reports (222)", networks)
	}
}

// TestWebNetworksRefuseWithoutTheCLI: a dashboard that was not given the CLI
// says so instead of reading the record behind the CLI's back.
func TestWebNetworksRefuseWithoutTheCLI(t *testing.T) {
	engine, _ := networksFixture(t, 111)
	if _, err := engine.Networks(context.Background()); err == nil || !strings.Contains(err.Error(), "chainbench CLI") {
		t.Fatalf("networks without the CLI gave %v, want a refusal naming the chainbench CLI", err)
	}
}

func (statusCLI) Run(context.Context, ...string) ([]byte, error) {
	return nil, errors.New("statusCLI answers chain status only")
}

// observeCLI answers `chain observe` by running the same use case the CLI
// runs, in this process, with the logins file the command names. Every other
// command is refused, so a test using it exercises observation only.
type observeCLI struct{}

func (observeCLI) ChainStatus(ctx context.Context, dir string) (State, error) {
	out, err := ChainStatus(ctx, Deps{}, ChainStatusIn{DataDir: dir})
	return out.State, err
}

func (observeCLI) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) < 2 || args[0] != "chain" || args[1] != "observe" {
		return nil, errors.New("observeCLI answers chain observe only")
	}
	flag := func(name string) string {
		for i, a := range args {
			if a == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	out, err := ChainObserve(ctx, Deps{}.WithLogins(flag("--server-logins")), ChainObserveIn{DataDir: flag("--workspace-dir")})
	if err != nil {
		return nil, err
	}
	return json.Marshal(out)
}
