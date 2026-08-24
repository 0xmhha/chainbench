package driver

import (
	"strings"
	"testing"
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
