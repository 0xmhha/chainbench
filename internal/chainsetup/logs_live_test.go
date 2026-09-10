package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testsupport"
)

// TestLive_LogsReadsANodeLogFromItsServer verifies that a node's log is read
// from ITS machine, not this one — the read failure-data collection relies on
// for a remote/docker run. Set up first:
//
//	docker exec chainbench-server1 sh -c \
//	  'mkdir -p /data/chainbench && printf "start\n%s\nlast-line\n" "$(seq 1 400)" > /data/chainbench/nodelog-test.log'
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/chainsetup -run TestLive_LogsReads -v
func TestLive_LogsReadsANodeLogFromItsServer(t *testing.T) {
	build := testsupport.ServersBuildDir(t)
	w, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.ServerSet = filepath.Join(build, "server-set.yaml")
	w.state.Docker = true
	w.state.Target = resource.Spec{DataRoot: "/data/chainbench"}
	w.state.Nodes = []node.Record{{
		Index: 1, Label: "node1", Server: "server1",
		LogPath: "/data/chainbench/nodelog-test.log",
	}}

	out, err := w.Logs(context.Background(), 1, 200)
	if err != nil {
		t.Fatalf("Logs (remote read): %v", err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 200 {
		t.Fatalf("tail has %d lines, want 200", len(lines))
	}
	if lines[len(lines)-1] != "last-line" {
		t.Fatalf("last line = %q, want last-line", lines[len(lines)-1])
	}
}
