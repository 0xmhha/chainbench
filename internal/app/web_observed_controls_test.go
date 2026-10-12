package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestObservedControlsFollowExecutablePermissionWithoutLosingLiveState(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := OpenDeploymentStore(filepath.Join(root, "store"))
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "observer", Role: "operator"}
	setInput := deploymentTestSet()
	setInput.Content = json.RawMessage(strings.ReplaceAll(string(setInput.Content), "localhost.", "127.0.0.1"))
	configInput := deploymentTestConfig()
	configInput.Content = json.RawMessage(strings.ReplaceAll(string(configInput.Content), "/data/chainbench", root))
	set, err := store.SaveDocument(actor, "", 0, setInput)
	if err != nil {
		t.Fatal(err)
	}
	config, err := store.SaveDocument(actor, "", 0, configInput)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := store.SaveWorkspace(actor, "", 0, DeploymentWorkspaceInput{Name: "owned observation", Documents: []DeploymentDocumentRef{{set.ID, set.Revision}, {config.ID, config.Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "owned-sleep")
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(binary, raw, 0700); err != nil {
		t.Fatal(err)
	}
	args := []string{"-test.run=^TestOwnedPermissionProcess$"}
	child := exec.Command(binary, args...)
	child.Env = append(os.Environ(), "CHAINBENCH_OWNED_PERMISSION_PROCESS=1")
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	spec := resource.Spec{DataRoot: root}
	target, err := (resource.Opener{}).Inspect(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	control := filepath.Join(root, "networks", workspace.ID)
	if err = os.MkdirAll(control, 0700); err != nil {
		t.Fatal(err)
	}
	state := State{Chain: "wbft", Binary: binary, Target: spec, Nodes: []node.Record{{Index: 1, Label: "node1", Role: "en", PID: child.Process.Pid, Args: args, DataDir: filepath.Join(root, "node1"), ConfigPath: filepath.Join(root, "node1.toml")}}}
	for name, value := range map[string]any{"chain-record.json": state, "web-target.json": target} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(control, name), encoded, 0600); err != nil {
			t.Fatal(err)
		}
	}
	engine := NewWebChainEngine(root, "", store, nil, nil, nil)
	recordBefore, err := os.ReadFile(filepath.Join(control, "chain-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	check := func(launchable bool) {
		t.Helper()
		network, err := engine.ObserveNetwork(ctx, actor, workspace.ID)
		if err != nil {
			t.Fatal(err)
		}
		n := network.Nodes[0]
		if n.State != "running" || n.ObservedPID != child.Process.Pid {
			args, _ := exec.Command("ps", "-ww", "-p", fmt.Sprint(child.Process.Pid), "-o", "args=", "-o", "comm=").Output()
			t.Fatal("permission inspection lost actual process", n, string(args))
		}
		if !slices.Contains(n.SupportedControls, "node.stop") {
			t.Fatal("live node explicit stop unavailable", n)
		}
		for _, operation := range []string{"node.start", "node.restart", "node.swap", "node.reset"} {
			if slices.Contains(n.SupportedControls, operation) != launchable {
				t.Errorf("execution permission ignored for %s: %+v", operation, n)
			}
		}
		if !launchable && n.ObservationReason != "binary_launch_unavailable" {
			t.Errorf("unavailable executable unexplained: %+v", n)
		}
		if launchable && n.ObservationReason != "" {
			t.Errorf("restored permission still unexplained: %+v", n)
		}
	}
	check(true)
	if err = os.Chmod(binary, 0600); err != nil {
		t.Fatal(err)
	}
	check(false)
	if err = os.Chmod(binary, 0700); err != nil {
		t.Fatal(err)
	}
	check(true)
	recordAfter, err := os.ReadFile(filepath.Join(control, "chain-record.json"))
	if err != nil || string(recordAfter) != string(recordBefore) {
		t.Fatal("read-only permission probe changed record", err)
	}
}

func TestOwnedPermissionProcess(t *testing.T) {
	if os.Getenv("CHAINBENCH_OWNED_PERMISSION_PROCESS") == "1" {
		time.Sleep(2 * time.Minute)
		os.Exit(0)
	}
}
