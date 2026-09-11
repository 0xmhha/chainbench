package process

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
	// The property is that mkdir is separated from nohup by `|| exit 1;` and not
	// by `&&`, not that the two are adjacent -- the attempt marker sits between
	// them now. Asserting adjacency pinned the spelling and would fail every time
	// something correct is added in that gap.
	if !strings.Contains(cmd, "|| exit 1;") {
		t.Fatalf("mkdir must be separated from the launch by `|| exit 1;`:\n%s", cmd)
	}
	if i, j := strings.Index(cmd, "|| exit 1;"), strings.Index(cmd, "nohup "); i < 0 || j < i {
		t.Fatalf("nohup must come after `|| exit 1;`:\n%s", cmd)
	}
	for _, want := range []string{">> '/data/logs/node1.log' 2>&1 < /dev/null &", "echo $!"} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("launch command lost %q:\n%s", want, cmd)
		}
	}
}

// TestLaunchCommand_AppendsSoARestartKeepsTheLastAttempt: `>` erased the previous
// attempt every time a node restarted, which is precisely when the log is the
// evidence wanted. The local driver has always used O_APPEND, so truncating here
// also made the two targets leave behind different things.
func TestLaunchCommand_AppendsSoARestartKeepsTheLastAttempt(t *testing.T) {
	cmd := launchCommand(NodeSpec{
		Index: 2, Binary: "/data/bin/gwbft",
		LogPath: "/data/logs/node2.log",
	})
	if strings.Contains(cmd, "> '/data/logs/node2.log'") && !strings.Contains(cmd, ">> '/data/logs/node2.log'") {
		t.Fatalf("the node's output truncates its log, so a restart erases the last attempt:\n%s", cmd)
	}
	// Every redirection at the log must append: the marker's and the node's.
	if n := strings.Count(cmd, ">> '/data/logs/node2.log'"); n != 2 {
		t.Fatalf("expected both the marker and the node output to append, got %d appends:\n%s", n, cmd)
	}
}

// TestLaunchCommand_MarksEachAttempt: one file now holds several attempts, so
// something has to say where one ends. The marker names the binary because a
// restart may be a different build -- that is what a hardfork swap is.
func TestLaunchCommand_MarksEachAttempt(t *testing.T) {
	cmd := launchCommand(NodeSpec{
		Index: 3, Binary: "/data/bin/gwemix",
		LogPath: "/data/logs/node3.log",
	})
	for _, want := range []string{"chainbench: launch node3", "gwemix"} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("the attempt marker should name %q:\n%s", want, cmd)
		}
	}
	// The marker is written before the node starts, or it labels the wrong output.
	if i, j := strings.Index(cmd, "chainbench: launch node3"), strings.Index(cmd, "nohup "); i < 0 || i > j {
		t.Fatalf("the marker must be written before the node is launched:\n%s", cmd)
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
