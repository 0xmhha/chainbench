package mcp_test

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestNodeResetTool_RefusesAProducer: the MCP mirror of `node reset` refuses a
// producer before stopping it, exactly as the CLI does.
func TestNodeResetTool_RefusesAProducer(t *testing.T) {
	bp := exec.Command("sleep", "30")
	if err := bp.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bp.Process.Kill(); _ = bp.Wait() })
	dir := t.TempDir()
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	label := node.LabelFor(1)
	layout := node.Layout{Root: dir}
	if err = comp.Save(chainsetup.State{
		FormatVersion: chainsetup.StateFormatVersion, Chain: "stablenet", BPCount: 1,
		Target: resource.Spec{DataRoot: dir}, Steps: map[string]chainsetup.Step{},
		Nodes: []node.Record{{Index: 1, Label: string(label), Role: "bp", Host: "127.0.0.1",
			DataDir: layout.DataDir(label), PID: bp.Process.Pid}},
	}); err != nil {
		t.Fatal(err)
	}

	text, isErr := callText(t, newServer(), "chainbench_node_reset", map[string]any{"workspaceDir": dir, "index": 1})
	if !isErr || !strings.Contains(text, "block producer") {
		t.Fatalf("reset of a producer gave %q (error %v), want a refusal naming the producer", text, isErr)
	}
	if syscall.Kill(bp.Process.Pid, 0) != nil {
		t.Error("the producer was stopped before the refusal")
	}
}

// TestNodeSwapTool_NeedsAWorkspaceAndIndex: the swap tool reaches the same use
// case as the CLI and reports its input error unchanged.
func TestNodeSwapTool_NeedsAWorkspaceAndIndex(t *testing.T) {
	text, isErr := callText(t, newServer(), "chainbench_node_swap", map[string]any{"binary": "/bin/true"})
	if !isErr || !strings.Contains(text, "a workspace directory and a 1-based node index are required") {
		t.Fatalf("swap without a workspace gave %q (error %v)", text, isErr)
	}
}
