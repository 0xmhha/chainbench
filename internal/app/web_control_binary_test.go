package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestWebControlBinaryAcceptsOnlyRegisteredOwnedExecutables(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "registered-source")
	bytes := []byte("verified binary fixture")
	if err := os.WriteFile(source, bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	config := deploymentTestConfig()
	var declaration map[string]any
	if err := json.Unmarshal(config.Content, &declaration); err != nil {
		t.Fatal(err)
	}
	declaration["dataRoot"] = root
	content, err := json.Marshal(declaration)
	if err != nil {
		t.Fatal(err)
	}
	config.Content = content
	payload := webChainPayload{ExecutionBinary: source, Binary: ManifestBinaryEvidence{Chain: "wbft", SHA256: manifestHash(bytes)}, Config: DeploymentDocument{DeploymentDocumentInput: config}}

	wc, err := deploymentWorkspace(payload.Config.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	native, err := wc.Resolve(resource.PurposeBinaries, payload.Binary.SHA256+"/gwbft")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(native), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(native, bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	state := State{Target: resource.Spec{DataRoot: root}}
	for _, binary := range []string{source, native} {
		state.Binary = binary
		got, err := bindWebControlBinary(context.Background(), state, payload, nil)
		if err != nil || got != binary {
			t.Fatalf("registered executable rejected: %s %s %v", binary, got, err)
		}
	}
	unrelated := filepath.Join(root, "unrelated")
	if err = os.WriteFile(unrelated, bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	state.Binary = unrelated
	if _, err = bindWebControlBinary(context.Background(), state, payload, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("same bytes at undeclared executable path accepted", err)
	}
	state.Binary = native
	if err = os.WriteFile(native, []byte("substituted bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err = bindWebControlBinary(context.Background(), state, payload, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("owned file replacement accepted", err)
	}
	if err = os.Remove(native); err != nil {
		t.Fatal(err)
	}
	if _, err = bindWebControlBinary(context.Background(), state, payload, nil); err == nil {
		t.Fatal("missing execution binary accepted")
	}
}

func TestWebReviewedExecutableCannotChangeItsProtocolOrGenesisMapping(t *testing.T) {
	root := t.TempDir()
	raw := []byte("reviewed executable fixture")
	source := filepath.Join(root, "base")
	if err := os.WriteFile(source, raw, 0700); err != nil {
		t.Fatal(err)
	}
	config := deploymentTestConfig()
	var declaration map[string]any
	if err := json.Unmarshal(config.Content, &declaration); err != nil {
		t.Fatal(err)
	}
	declaration["dataRoot"] = root
	config.Content, _ = json.Marshal(declaration)
	evidence := ManifestBinaryEvidence{ID: "reviewed", Chain: "wbft", SHA256: manifestHash(raw), OS: runtime.GOOS, Architecture: runtime.GOARCH}
	p := webChainPayload{Input: WebPlanInput{NodeIDs: []string{"node1"}}, ExecutionBinary: source, Binary: evidence, Target: resource.Inspection{OS: runtime.GOOS, Architecture: runtime.GOARCH}, Config: DeploymentDocument{DeploymentDocumentInput: config}}
	target, err := webNodeBinaryTarget(p, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(target, raw, 0700); err != nil {
		t.Fatal(err)
	}
	p.CurrentNodeBinary = &webNodeBinary{NodeID: "node1", Evidence: evidence, Path: target}
	state := State{Chain: "wbft", Binary: source, GenesisPath: "/reviewed/genesis", Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{{Index: 1, Label: "node1", Binary: "node1"}}, Binaries: map[string]string{"node1": target}}
	if _, err = bindWebControlBinary(context.Background(), state, p, nil); err != nil {
		t.Fatal("valid reviewed executable refused", err)
	}

	p.Input.Operation = "node.restart"
	if err = os.Chmod(target, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = bindWebControlBinary(context.Background(), state, p, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("reviewed named nonexecutable target accepted", err)
	}
	p.Input.Operation = "node.stop"
	if _, err = bindWebControlBinary(context.Background(), state, p, nil); err != nil {
		t.Fatal("named target execute permission prevents explicit stop", err)
	}
	if err = os.Chmod(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(source, 0600); err != nil {
		t.Fatal(err)
	}
	p.Input.Operation = "node.restart"
	if _, err = bindWebControlBinary(context.Background(), state, p, nil); err != nil {
		t.Fatal("unused base execution permission prevents named relaunch", err)
	}
	for _, field := range []string{"chain", "genesis", "genesis-config"} {
		changed := state
		switch field {
		case "chain":
			changed.BinaryChains = map[string]string{"node1": "wemix"}
		case "genesis":
			changed.GenesisPaths = map[string]string{"node1": "/unreviewed/genesis"}
		case "genesis-config":
			changed.GenesisConfigPaths = map[string]string{"node1": "/unreviewed/config"}
		}
		if _, err = bindWebControlBinary(context.Background(), changed, p, nil); !errors.Is(err, ErrDeploymentConflict) {
			t.Fatal("reviewed bytes authorized unreviewed node declarations", field, err)
		}
	}
}

func TestWebControlBinaryRejectsUnboundSelectedNodeExecutables(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "registered")
	raw := []byte("registered executable fixture")
	if err := os.WriteFile(path, raw, 0700); err != nil {
		t.Fatal(err)
	}
	p := webChainPayload{Input: WebPlanInput{NodeIDs: []string{"node1"}}, ExecutionBinary: path, Binary: ManifestBinaryEvidence{Chain: "wbft", SHA256: manifestHash(raw)}}
	state := State{Binary: path, Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{{Index: 1, Label: "node1"}, {Index: 2, Label: "node2"}}, Binaries: map[string]string{"unreviewed": path}}
	for _, name := range []string{"unreviewed", "missing"} {
		state.Nodes[0].Binary = name
		if _, err := bindWebControlBinary(context.Background(), state, p, nil); !errors.Is(err, ErrDeploymentConflict) {
			t.Fatal("unbound selected executable accepted", name, err)
		}
	}
	state.Nodes[0].Binary = ""
	state.Nodes[1].Binary = "unreviewed"
	if got, err := bindWebControlBinary(context.Background(), state, p, nil); err != nil || got != path {
		t.Fatal("unselected binary prevents controlling the base node", got, err)
	}
}

func TestWebRelaunchRefusesNonExecutableBytesButAllowsExplicitStop(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "registered")
	raw := []byte("reviewed bytes without execute permission")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	p := webChainPayload{Input: WebPlanInput{NodeIDs: []string{"node1"}}, ExecutionBinary: path, Binary: ManifestBinaryEvidence{Chain: "wbft", SHA256: manifestHash(raw)}}
	state := State{Binary: path, Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{{Index: 1, Label: "node1"}}}
	for _, operation := range []string{"node.start", "node.restart", "node.swap", "node.reset"} {
		p.Input.Operation = operation
		if _, err := bindWebControlBinary(context.Background(), state, p, nil); !errors.Is(err, ErrDeploymentConflict) {
			t.Errorf("nonexecutable target accepted for %s: %v", operation, err)
		}
	}
	p.Input.Operation = "node.stop"
	if _, err := bindWebControlBinary(context.Background(), state, p, nil); err != nil {
		t.Fatal("removing execute permission prevents explicit stop", err)
	}
	if err := os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	p.Input.Operation = "node.restart"
	if _, err := bindWebControlBinary(context.Background(), state, p, nil); err != nil {
		t.Fatal("restored executable remains blocked", err)
	}
}
