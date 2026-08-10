package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNetCmd_ComposeStepByStep(t *testing.T) {
	dir := t.TempDir()
	presetDir := filepath.Join("..", "..", "keys", "preset")

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
