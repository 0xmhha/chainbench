package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestWebNodeResetRechecksProcessBeforeEffects(t *testing.T) {
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
	state, payload := webResetFixture(t)
	state.Chain, state.Binary = "wbft", "/bin/sleep"
	state.Nodes[0].PID = child.Process.Pid
	state.Nodes[0].Args = []string{"--config", state.Nodes[0].ConfigPath}

	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(control, "chain-record.json")
	if err = os.WriteFile(record, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	checksum, err := (filestore.Local{}).Checksum(context.Background(), "/bin/sleep")
	if err != nil {
		t.Fatal(err)
	}
	payload.ExecutionBinary, payload.ControlDir, payload.RecordDigest = "/bin/sleep", control, manifestHash(raw)
	payload.Binary = ManifestBinaryEvidence{Chain: "wbft", SHA256: strings.TrimPrefix(checksum, "sha256:")}

	_, err = engine.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, payload, func(WebJobPhase) error { return nil })
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("unverified process reached the node reset verb: %v", err)
	}
	if !process.Alive(child.Process.Pid) {
		t.Fatal("foreign fixture process was stopped")
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(raw) {
		t.Fatal("rejected control mutated the record", err)
	}
}

func TestWebTestInputsWithoutGenesisSurviveAcceptedSnapshot(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"nodes":[{"index":1,"role":"en"},{"index":2,"role":"bp"}]}`)
	if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var accepted webChainPayload
	if err = json.Unmarshal(raw, &accepted); err != nil {
		t.Fatal(err)
	}
	if err = e.recheckTestGenesis(context.Background(), accepted); err != nil {
		t.Fatalf("unchanged generated genesis case fails after plan persistence: %v", err)
	}
}

// webResetFixture provides valid owned paths and immutable reset inputs only.
func webResetFixture(t *testing.T) (State, webChainPayload) {
	t.Helper()
	root := t.TempDir()
	config := deploymentTestConfig()
	config.Content = json.RawMessage(strings.Replace(string(config.Content), "/data/chainbench", root, 1))
	set := deploymentTestSet()
	wc, err := deploymentWorkspace(config)
	if err != nil {
		t.Fatal(err)
	}
	layout := node.Layout{Root: root, CompositionID: "abcdef123456", NodesDir: wc.Paths.Nodes, RuntimeDir: wc.Paths.Runtime, LogsDir: wc.Paths.Logs}
	ns := node.Record{Index: 1, Role: "en", PID: 1234, DataDir: layout.DataDir("node1"), ConfigPath: layout.ConfigPath("node1")}
	if err = os.MkdirAll(ns.DataDir, 0700); err != nil {
		t.Fatal(err)
	}
	state := State{FormatVersion: 3, CompositionID: layout.CompositionID, Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{ns}, WorkspaceConfig: filepath.Join(root, "workspace-config.yaml"), ServerSet: filepath.Join(root, "server-set.yaml"), GenesisPath: filepath.Join(root, wc.Paths.Runtime, layout.CompositionID, "genesis.json"), LaunchInputs: map[string]string{}}
	p := webChainPayload{Input: WebPlanInput{Operation: "node.reset", NodeIDs: []string{"node1"}}, Config: DeploymentDocument{DeploymentDocumentInput: config}, Set: DeploymentDocument{DeploymentDocumentInput: set}}
	for _, entry := range []struct {
		path string
		doc  DeploymentDocument
	}{{state.WorkspaceConfig, p.Config}, {state.ServerSet, p.Set}} {
		raw, err := ExportDeploymentDocument(entry.doc.DeploymentDocumentInput, "yaml")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(entry.path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{ns.ConfigPath, state.GenesisPath} {
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		raw := []byte("immutable reset input")
		if err = os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		state.LaunchInputs[path] = "sha256:" + manifestHash(raw)
	}
	p.Target, err = (resource.Opener{}).Inspect(context.Background(), state.Target)
	if err != nil {
		t.Fatal(err)
	}
	return state, p
}

func TestWebResetSelectionRefusesProducersAndUnownedPaths(t *testing.T) {
	for _, name := range []string{"endpoint", "proxy", "producer", "unknown role", "no PID", "foreign binary", "outside data", "outside config", "traversal", "missing node", "multiple nodes"} {
		t.Run(name, func(t *testing.T) {
			state, p := webResetFixture(t)
			switch name {
			case "proxy":
				state.Nodes[0].Role = "pn"
			case "producer":
				state.Nodes[0].Role = "bp"
			case "unknown role":
				state.Nodes[0].Role = "unclassified"
			case "no PID":
				state.Nodes[0].PID = 0
			case "foreign binary":
				state.Nodes[0].Binary = "unregistered"
			case "outside data":
				state.Nodes[0].DataDir = filepath.Dir(state.Nodes[0].DataDir)
			case "outside config":
				state.Nodes[0].ConfigPath = "/outside/config.toml"
			case "traversal":
				state.CompositionID = "../outside"
			case "missing node":
				p.Input.NodeIDs = []string{"node2"}
			case "multiple nodes":
				p.Input.NodeIDs = []string{"node1", "node2"}
			}
			_, err := webResetNode(state, p)
			if name == "endpoint" || name == "proxy" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("unsafe reset selection accepted")
			}
			if name == "producer" && !strings.Contains(err.Error(), "producer") {
				t.Fatal(err)
			}
		})
	}
}

func TestWebResetRechecksFilesAndPhysicalDirectory(t *testing.T) {
	for _, name := range []string{"unchanged", "genesis", "config", "workspace", "server-set", "symlink"} {
		t.Run(name, func(t *testing.T) {
			state, p := webResetFixture(t)
			marker := filepath.Join(state.Nodes[0].DataDir, "old-data")
			if err := os.WriteFile(marker, []byte("retain until accepted"), 0600); err != nil {
				t.Fatal(err)
			}
			paths := map[string]string{"genesis": state.GenesisPath, "config": state.Nodes[0].ConfigPath, "workspace": state.WorkspaceConfig, "server-set": state.ServerSet}
			if path := paths[name]; path != "" {
				if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "symlink" {
				original := state.Nodes[0].DataDir
				outside := t.TempDir()
				if err := os.Rename(original, filepath.Join(outside, "node")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "node"), original); err != nil {
					t.Fatal(err)
				}
			}
			err := verifyWebResetInputs(context.Background(), state, p, nil)
			if name == "unchanged" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrDeploymentConflict) {
				t.Fatal("modified reset input accepted", err)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatal("input verification changed node data", err)
			}
		})
	}
}
