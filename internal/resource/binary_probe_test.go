package resource

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// localRunner gives the probe a real shell, so the quoting and the output
// framing are exercised rather than assumed.
func localRunner(t *testing.T) *Access {
	t.Helper()
	return &Access{Runner: func(ctx context.Context, command string) (remote.ExecResult, error) {
		cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("command: %v", err)
		}
		return remote.ExecResult{Stdout: string(out)}, nil
	}}
}

func writeProbeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "program")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBinaryProbe_ReturnsOutputAndHidesTheHome(t *testing.T) {
	path := writeProbeScript(t, `echo "datadir $HOME/data"; echo "arg:$1"; echo oops >&2; exit 3`)
	got, err := runBinaryProbe(context.Background(), localRunner(t), ProbeRequest{Path: path, Args: []string{"a b'c"}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got.Output)
	if got.ExitCode != 3 || !strings.Contains(text, "datadir $HOME/data") || !strings.Contains(text, "arg:a b'c") || !strings.Contains(text, "oops") {
		t.Fatalf("result = %d %q", got.ExitCode, text)
	}
}

func TestBinaryProbe_GivesTheBinaryNoInheritedEnvironment(t *testing.T) {
	t.Setenv("PROBE_LEAK", "inherited")
	path := writeProbeScript(t, `echo "[$PROBE_LEAK]"`)
	got, err := runBinaryProbe(context.Background(), localRunner(t), ProbeRequest{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got.Output)) != "[]" {
		t.Fatalf("output = %q", got.Output)
	}
}

func TestBinaryProbe_RefusesALocalTarget(t *testing.T) {
	if _, err := runBinaryProbe(context.Background(), &Access{}, ProbeRequest{Path: "/x"}); err == nil {
		t.Fatal("a local target was probed")
	}
}

func TestParseProbe_RefusesMalformedOutput(t *testing.T) {
	for _, out := range []string{"", "/tmp/x", "/tmp/x\nnot-a-status\n"} {
		if _, err := parseProbe(out); err == nil {
			t.Errorf("accepted %q", out)
		}
	}
}

// A probe runs a program, so it must not reach any program on the machine.
func TestProbe_RunsOnlyFilesUnderTheDataRoot(t *testing.T) {
	target := Spec{DataRoot: "/data/chainbench", Server: "s1"}
	for _, path := range []string{"/bin/sh", "/data/chainbench", "/data/chainbench/../../bin/sh", "/data/chainbench2/x", "relative", "/data/chainbench/bin/x\ny", "/data/chainbench/a=b"} {
		if err := validateProbe(ProbeRequest{Target: target, Path: path}); err == nil {
			t.Errorf("accepted %q", path)
		}
	}
	if err := validateProbe(ProbeRequest{Target: Spec{DataRoot: "..", Server: "s1"}, Path: "../../bin/sh"}); err == nil {
		t.Error("a relative data root was accepted")
	}
	if err := validateProbe(ProbeRequest{Target: target, Path: "/data/chainbench/bin/abc/probe", Args: []string{"--help"}}); err != nil {
		t.Fatal(err)
	}
}

// The read-only tools decode their request strictly into ReadRequest. If a
// probe ever fits there, a tool declared read-only runs programs.
func TestReadRequest_CannotCarryAProbe(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"target":{"dataRoot":"/d"},"probe":{"path":"/bin/sh"}}`))
	decoder.DisallowUnknownFields()
	var request ReadRequest
	if err := decoder.Decode(&request); err == nil {
		t.Fatal("a read request accepted a probe")
	}
}
