package chaincmd_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/chaincmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// run executes a net command line the way an operator types it, on a bare
// root that mounts exactly what the real root mounts.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(chaincmd.New())
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

	out, err := run(t, "chain", "new", "--workspace-dir", dir, "--chain", "stablenet", "--keys", presetDir)
	if err != nil {
		t.Fatalf("chain new: %v\n%s", err, out)
	}
	if !strings.Contains(out, "stablenet") || !strings.Contains(out, "chain id 8283") ||
		!strings.Contains(out, "target local") || !strings.Contains(out, "keys "+presetDir) {
		t.Fatalf("chain new output: %s", out)
	}

	out, err = run(t, "chain", "status", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("chain status: %v\n%s", err, out)
	}
	for _, want := range []string{"chain: stablenet", "target: local", "keys: " + presetDir, "new", "true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("chain status missing %q:\n%s", want, out)
		}
	}
}

func TestNetCmd_RemoteTargetRecorded(t *testing.T) {
	dir := t.TempDir()
	out, err := run(t, "chain", "new", "--workspace-dir", dir, "--chain", "stablenet",
		"--remote-host", "10.0.0.1", "--remote-user", "ubuntu", "--target-dir", "/tmp/net")
	if err != nil {
		t.Fatalf("chain new remote: %v\n%s", err, out)
	}
	if !strings.Contains(out, "remote ubuntu@10.0.0.1:/tmp/net") {
		t.Fatalf("remote target not recorded: %s", out)
	}
}

// TestNetCmd_DefaultsTheWorkspace pins the omitted-flag behaviour: a fresh
// timestamped directory under the (test-scoped) home, and the path printed
// before anything uses it, so it is never a guess.
func TestNetCmd_DefaultsTheWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	out, err := run(t, "chain", "new", "--chain", "stablenet")
	if err != nil {
		t.Fatalf("chain new without --workspace-dir: %v\n%s", err, out)
	}
	if !strings.Contains(out, "workspace: "+home) {
		t.Fatalf("the default path was not announced:\n%s", out)
	}
	if !strings.Contains(out, "chainsetup") {
		t.Fatalf("default path missing the chainsetup segment:\n%s", out)
	}
}

// TestNetNew_RecordsTheServerSetWithDocker pins the pair travelling together:
// --docker names how servers are reached, --server-set names which exist, and
// a workspace told both at new time carries both to every later step.
func TestNetNew_RecordsTheServerSetWithDocker(t *testing.T) {
	dir := t.TempDir()
	set := filepath.Join(t.TempDir(), "server-set.yaml")
	if err := os.WriteFile(set, []byte(
		"version: 2\npool:\n  hosts: [{name: box1, addr: 192.0.2.11}]\n"+
			"ssh: {user: dev, password: pw}\ndataRoot: /data/cb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "chain", "new", "--workspace-dir", dir, "--chain", "stablenet",
		"--keys", filepath.Join("..", "..", "..", "keys", "preset"),
		"--docker", "--server-set", set); err != nil {
		t.Fatalf("chain new: %v", err)
	}
	out, err := run(t, "chain", "status", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("chain status: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "workspace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), set) {
		t.Errorf("workspace did not record the server set:\n%s\nstatus:\n%s", raw, out)
	}
}
