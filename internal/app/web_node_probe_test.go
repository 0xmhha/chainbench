package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

type nodeProbeFixture struct {
	alive  []bool
	argv   []string
	err    error
	probes int
}

func (f *nodeProbeFixture) PIDAlive(context.Context, int) (bool, error) {
	i := f.probes
	f.probes++
	if f.err != nil {
		return false, f.err
	}
	if i >= len(f.alive) {
		return false, nil
	}
	return f.alive[i], nil
}
func (f *nodeProbeFixture) Cmdline(context.Context, int) ([]string, error) { return f.argv, f.err }

func TestWebNodeProbeVerifiesRecordedProcess(t *testing.T) {
	ns := node.Record{Index: 1, PID: 1234, DataDir: "/owned/node1", ConfigPath: "/owned/node1.toml", Args: []string{"--config", "/owned/node1.toml"}}
	for _, tc := range []struct {
		name          string
		fixture       nodeProbeFixture
		pid           int
		state, reason string
	}{
		{name: "exact launch", fixture: nodeProbeFixture{alive: []bool{true, true}, argv: []string{"/owned/gwbft", "--config", "/owned/node1.toml"}}, pid: 1234, state: "running"},
		{name: "missing recorded process", fixture: nodeProbeFixture{alive: []bool{false}}, pid: 1234, state: "missing", reason: "recorded_pid_absent"},
		{name: "PID reused for different binary", fixture: nodeProbeFixture{alive: []bool{true}, argv: []string{"/bin/sleep", "30"}}, pid: 1234, state: "ownership_mismatch", reason: "argv_mismatch"},
		{name: "same binary different owned config", fixture: nodeProbeFixture{alive: []bool{true}, argv: []string{"/owned/gwbft", "--config", "/other/node1.toml"}}, pid: 1234, state: "ownership_mismatch", reason: "argv_mismatch"},
		{name: "same binary incomplete args", fixture: nodeProbeFixture{alive: []bool{true}, argv: []string{"/owned/gwbft"}}, pid: 1234, state: "ownership_mismatch", reason: "argv_mismatch"},
		{name: "process exits during read", fixture: nodeProbeFixture{alive: []bool{true, false}, argv: []string{"/owned/gwbft", "--config", "/owned/node1.toml"}}, pid: 1234, state: "missing", reason: "recorded_pid_absent"},
		{name: "unavailable probe", fixture: nodeProbeFixture{err: errors.New("private-probe-error")}, pid: 1234, state: "unknown", reason: "probe_unavailable"},
		{name: "no recorded PID", pid: 0, state: "unknown", reason: "no_recorded_pid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := ns
			n.PID = tc.pid
			before := n
			before.Args = append([]string(nil), n.Args...)
			got := observeWebNodeProcess(context.Background(), &tc.fixture, "/owned/gwbft", n)
			if got.State != tc.state || got.Reason != tc.reason {
				t.Fatal(got)
			}
			if got.State == "running" && got.PID != n.PID {
				t.Fatal("lost live PID")
			}
			if got.State != "running" && got.PID != 0 {
				t.Fatal("unverified PID presented as live")
			}
			if !reflect.DeepEqual(n, before) {
				t.Fatal("read-only probe mutated the owned record")
			}
		})
	}
}

func TestWebNodeExecutionRechecksProcessBeforeStopping(t *testing.T) {
	child := exec.Command("/bin/sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewWebChainEngine(root, "", store, nil, nil, nil)
	control := filepath.Join(root, "network")
	if err = os.Mkdir(control, 0o700); err != nil {
		t.Fatal(err)
	}
	state := State{FormatVersion: 3, Chain: "wbft", Binary: "/owned/gwbft", Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{{Index: 1, PID: child.Process.Pid, Role: "en", DataDir: filepath.Join(root, "node1"), Args: []string{"--config", filepath.Join(root, "node1.toml")}}}}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(control, "chain-record.json")
	if err = os.WriteFile(record, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	payload := webChainPayload{ControlDir: control, RecordDigest: manifestHash(raw), Input: WebPlanInput{Operation: "node.stop", NodeIDs: []string{"node1"}}, Set: DeploymentDocument{DeploymentDocumentInput: DeploymentDocumentInput{Kind: "server-set", ContractVersion: "2", Content: json.RawMessage(`{"version":2,"pool":{"hosts":[{"name":"local","addr":"127.0.0.1"}],"slots":1}}`)}}}
	_, err = engine.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, payload, func(WebJobPhase) error { return nil })
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("unverified process reached the node stop verb: %v", err)
	}
	if !process.Alive(child.Process.Pid) {
		t.Fatal("foreign fixture process was stopped")
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(raw) {
		t.Fatal("rejected control mutated the record", err)
	}
}
