package testengine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestGatherFailureData_KeepsTheWholeLog: a failure keeps each node's log as the
// node wrote it. Only the first and last 200 lines used to be kept, so a cause
// in the middle — here the one line that says the node fell out of sync — was
// never in the evidence.
func TestGatherFailureData_KeepsTheWholeLog(t *testing.T) {
	dir := t.TempDir()
	label := node.LabelFor(1)
	layout := node.Layout{Root: dir}
	logPath := layout.LogPath(label)

	lines := make([]string, 0, 1000)
	for i := 1; i <= 1000; i++ {
		line := fmt.Sprintf("INFO imported block %d", i)
		if i == 500 {
			line = "WARN Synchronisation failed, dropping peer"
		}
		lines = append(lines, line)
	}
	whole := strings.Join(lines, "\n")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte(whole+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := chainsetup.State{
		FormatVersion: chainsetup.StateFormatVersion,
		Chain:         "stablenet",
		Target:        resource.Spec{DataRoot: dir},
		Nodes: []node.Record{{
			Index: 1, Label: string(label), Role: string(node.RoleBP), Host: "127.0.0.1",
			DataDir: layout.DataDir(label), ConfigPath: layout.ConfigPath(label), LogPath: logPath,
		}},
		Steps: map[string]chainsetup.Step{},
	}
	if err := comp.Save(st); err != nil {
		t.Fatal(err)
	}

	var got string
	for _, e := range gatherFailureData(context.Background(), chainsetup.Deps{}, dir, nil) {
		if e.Name == "node1.log" {
			got = string(e.Data)
		}
	}
	if got != whole+"\n" {
		t.Fatalf("node1.log holds %d of the log's %d lines", strings.Count(got, "\n"), len(lines))
	}
}
