package netcmd_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/netcmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// run executes a net command line the way an operator types it, on a bare
// root that mounts exactly what the real root mounts.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(netcmd.New())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

func TestNetCmd_ComposeStepByStep(t *testing.T) {
	dir := t.TempDir()
	presetDir := filepath.Join("..", "..", "..", "keys", "preset")

	out, err := run(t, "net", "new", "--data-dir", dir, "--chain", "stablenet", "--keys", presetDir)
	if err != nil {
		t.Fatalf("net new: %v\n%s", err, out)
	}
	if !strings.Contains(out, "stablenet") || !strings.Contains(out, "chain id 8283") ||
		!strings.Contains(out, "target local") || !strings.Contains(out, "keys "+presetDir) {
		t.Fatalf("net new output: %s", out)
	}

	out, err = run(t, "net", "status", "--data-dir", dir)
	if err != nil {
		t.Fatalf("net status: %v\n%s", err, out)
	}
	for _, want := range []string{"chain: stablenet", "target: local", "keys: " + presetDir, "new", "true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("net status missing %q:\n%s", want, out)
		}
	}
}

func TestNetCmd_RemoteTargetRecorded(t *testing.T) {
	dir := t.TempDir()
	out, err := run(t, "net", "new", "--data-dir", dir, "--chain", "stablenet",
		"--remote-host", "10.0.0.1", "--remote-user", "ubuntu", "--target-dir", "/tmp/net")
	if err != nil {
		t.Fatalf("net new remote: %v\n%s", err, out)
	}
	if !strings.Contains(out, "remote ubuntu@10.0.0.1:/tmp/net") {
		t.Fatalf("remote target not recorded: %s", out)
	}
}

func TestNetCmd_RequiresDataDir(t *testing.T) {
	if _, err := run(t, "net", "new", "--chain", "stablenet"); err == nil {
		t.Fatal("expected error without --data-dir")
	}
}
