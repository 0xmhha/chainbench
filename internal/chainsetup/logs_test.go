package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// wsWithLocalLog builds a local-target workspace with one node whose log is the
// given local file, so Logs reads it back through the (local) machine.
func wsWithLocalLog(t *testing.T, logBody string) *Workspace {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "node1.log")
	if err := os.WriteFile(logPath, []byte(logBody), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Target = resource.Spec{DataRoot: dir}
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", LogPath: logPath}}
	return w
}

func TestLogs_TailsToLineCount(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString("line\n")
	}
	w := wsWithLocalLog(t, b.String())

	out, err := w.Logs(context.Background(), 1, 200)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if got := len(strings.Split(out, "\n")); got != 200 {
		t.Fatalf("tail has %d lines, want 200", got)
	}
}

func TestLogs_ShorterThanTailReturnsAll(t *testing.T) {
	w := wsWithLocalLog(t, "a\nb\nc\n")
	out, err := w.Logs(context.Background(), 1, 200)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if out != "a\nb\nc" {
		t.Fatalf("out = %q, want a\\nb\\nc", out)
	}
}

func TestLogs_UnknownNodeErrors(t *testing.T) {
	w := wsWithLocalLog(t, "x\n")
	if _, err := w.Logs(context.Background(), 9, 10); err == nil {
		t.Fatal("an unknown node index must error")
	}
}
