package filecmd_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/filecmd"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(filecmd.New())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// localTarget writes a server set naming this machine and a workspace config
// whose data root is a fresh directory.
func localTarget(t *testing.T) (set, config, root string) {
	t.Helper()
	dir := t.TempDir()
	root = filepath.Join(dir, "data")
	set, config = filepath.Join(dir, "server-set.yaml"), filepath.Join(dir, "workspace-config.yaml")
	if err := os.WriteFile(set, []byte("version: 2\npool:\n  hosts:\n    - {name: here, addr: 127.0.0.1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("version: 1\ndataRoot: "+root+"\npaths: {binaries: binaries, configs: configs, genesis: genesis, keystore: keystore, keyrings: keyrings, nodes: nodes, runtime: runtime, logs: logs}\ncontrol: {artifactRoot: out}\ninputs: {mode: generated}\nexecution: {chain: fresh}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return set, config, root
}

// TestUpload_SHA256ChecksTheBytesBeforeAndAfter: --sha256 refuses a local file
// that is not the expected bytes without writing anything, and on success the
// written file is those bytes.
func TestUpload_SHA256ChecksTheBytesBeforeAndAfter(t *testing.T) {
	set, config, root := localTarget(t)
	local := filepath.Join(t.TempDir(), "gstable")
	// A stored asset is private to its owner; the target needs it runnable.
	if err := os.WriteFile(local, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("#!/bin/sh\nexit 0\n"))
	dst := filepath.Join(root, "binaries", "abc", "gstable")
	base := []string{"file", "upload", local, "--server-set", set, "--server", "here", "--workspace-config", config, "--remote", dst, "--force-upload"}

	if _, err := run(t, append(base, "--sha256", strings.Repeat("0", 64))...); err == nil || !strings.Contains(err.Error(), "sha256") || strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("a wrong digest gave %v, want a refusal naming sha256", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("bytes that failed the digest were written")
	}
	if out, err := run(t, append(base, "--sha256", hex.EncodeToString(sum[:]), "--mode", "0755")...); err != nil {
		t.Fatalf("upload: %v\n%s", err, out)
	}
	if info, err := os.Stat(dst); err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("uploaded binary: %v %v", info, err)
	}
}

// TestUpload_ServerLoginsIsAccepted: an upload takes its logins apart from the
// server set, as run and chain up do.
func TestUpload_ServerLoginsIsAccepted(t *testing.T) {
	set, config, root := localTarget(t)
	local := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(local, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "file", "upload", local, "--server-set", set, "--server", "here", "--workspace-config", config,
		"--remote", filepath.Join(root, "f"), "--server-logins", set); err != nil {
		t.Fatalf("--server-logins: %v", err)
	}
}
