package testengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// A node's log is where its record says, which in a workspace with its own
// data root is not under the control directory. Reading the conventional path
// there found nothing, so a failed launch gave no reason.
func TestWorkspaceNodesLogReadsTheRecordedLogPath(t *testing.T) {
	control, data := t.TempDir(), t.TempDir()
	logPath := filepath.Join(data, "logs", "node5.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("INFO starting\nFatal: Error starting protocol stack: bind: can't assign requested address\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := chainsetup.State{FormatVersion: chainsetup.StateFormatVersion, Chain: "stablenet", Nodes: []node.Record{{Index: 5, Label: "node5", Role: "en", Host: "127.0.0.1", LogPath: logPath, DataDir: filepath.Join(data, "node5")}}}
	raw, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(control, "chain-record.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := workspaceNodes{dataDir: control}.Log(context.Background(), node.Node{Index: 5}, 64)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "can't assign requested address") || len(got) > 64 {
		t.Fatalf("log tail = %q", got)
	}
}
