package mcp_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// TestStopTool launches a real short-lived process, records its PID in a
// nodeset, and verifies the stop tool terminates it. This exercises the real
// LocalDriver stop path without needing a chain binary.
func TestStopTool(t *testing.T) {
	proc := exec.Command("sleep", "30")
	if err := proc.Start(); err != nil {
		t.Skipf("cannot start sleep: %v", err)
	}
	pid := proc.Process.Pid
	// reap the process so it does not linger if the assertion below fails.
	t.Cleanup(func() { _ = proc.Process.Kill(); _, _ = proc.Process.Wait() })

	dir := t.TempDir()
	writeWorkspace(t, dir, "stablenet", wsNode(dir, 1, "validator", 8501, pid))

	text, isErr := callText(t, newServer(), "chainbench_stop", map[string]any{"workspaceDir": dir})
	if isErr || !strings.Contains(text, "stopped 1 node") {
		t.Fatalf("stop: err=%v text=%s", isErr, text)
	}
	// the process should now be gone: Wait returns promptly.
	done := make(chan error, 1)
	go func() { _, err := proc.Process.Wait(); done <- err }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("process still running after stop")
	}
}

// writeWorkspace records a composed network the way the steps leave one.
func writeWorkspace(t *testing.T, dir, chain string, nodes ...node.Record) {
	t.Helper()
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := chainsetup.State{
		Chain: chain, Binary: "/opt/fakebin", Validators: len(nodes),
		Target: machine.Spec{DataRoot: dir}, Nodes: nodes,
		Capabilities: []string{"rpc"},
		Steps:        map[string]chainsetup.Step{},
	}
	if err := comp.Save(st); err != nil {
		t.Fatalf("write workspace: %v", err)
	}
}

// wsNode is one recorded node on this machine.
func wsNode(root string, index int, role string, http, pid int) node.Record {
	label := node.LabelFor(index)
	layout := node.Layout{Root: root}
	return node.Record{
		Index: index, Label: string(label), Role: role, Host: "127.0.0.1",
		DataDir: layout.DataDir(label), ConfigPath: layout.ConfigPath(label), LogPath: layout.LogPath(label),
		Endpoints: node.Endpoints{P2P: 30300 + index, HTTP: http},
		PID:       pid,
	}
}
