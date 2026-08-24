package driver

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// TestLaunchCommand_BackgroundsOnlyTheNode pins the shell grammar the remote
// launch depends on. With `mkdir && nohup CMD … &` the whole list is
// backgrounded: a subshell keeps the SSH session's pipes open while it waits
// on the node, and Launch hangs the first time a node survives its start.
// The command must background the node alone (`|| exit 1;` before nohup) and
// detach every stdio stream from the session.
func TestLaunchCommand_BackgroundsOnlyTheNode(t *testing.T) {
	cmd := launchCommand(NodeSpec{
		Index:   1,
		Binary:  "/data/bin/gstable",
		Args:    []string{"--datadir", "/data/node1"},
		LogPath: "/data/logs/node1.log",
	})

	if strings.Contains(cmd, "&& nohup") {
		t.Fatalf("mkdir joined to nohup with && backgrounds the whole list:\n%s", cmd)
	}
	for _, want := range []string{"|| exit 1; nohup", "> '/data/logs/node1.log' 2>&1 < /dev/null &", "echo $!"} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("launch command lost %q:\n%s", want, cmd)
		}
	}
}

// TestSudoWrap_KeepsThePasswordOffTheCommandLine pins the sudo shaping: the
// password travels on stdin (-S), never in the line; every command
// re-authenticates (-k); and the inner command survives quoting intact.
func TestSudoWrap_KeepsThePasswordOffTheCommandLine(t *testing.T) {
	cmd := sudoWrap(`mkdir -p '/data/it''s here' && echo done`)
	for _, want := range []string{"sudo -S -k -p ''", "/bin/sh -c "} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("sudo wrap lost %q:\n%s", want, cmd)
		}
	}
	if strings.Contains(cmd, "chainbench\n") || strings.Contains(cmd, "--password") {
		t.Fatalf("a password reached the command line:\n%s", cmd)
	}
}

// TestProbePorts_AsksBothFacesFromTheTarget pins the remote occupancy probe:
// the question runs ON the target (bash /dev/tcp), tries loopback AND the
// host's own address — a listener may be bound to either — and parses only
// the ports that accepted.
func TestProbePorts_AsksBothFacesFromTheTarget(t *testing.T) {
	var got string
	d := NewRemoteDriver(func(_ context.Context, cmd string) (remote.ExecResult, error) {
		got = cmd
		return remote.ExecResult{Stdout: "8600\n31000\n"}, nil
	})
	open, err := d.ProbePorts(context.Background(), "172.30.0.11", []int{8600, 8610, 31000})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/dev/tcp/127.0.0.1/$p", "/dev/tcp/172.30.0.11/$p", "8600 8610 31000"} {
		if !strings.Contains(got, want) {
			t.Fatalf("probe script lost %q:\n%s", want, got)
		}
	}
	if len(open) != 2 || open[0] != 8600 || open[1] != 31000 {
		t.Fatalf("open = %v, want [8600 31000]", open)
	}
}
