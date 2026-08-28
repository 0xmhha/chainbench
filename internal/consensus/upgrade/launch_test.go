package upgrade_test

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
)

func TestLaunchArgs(t *testing.T) {
	n := upgrade.NodeSpec{
		NetworkID: 8285,
		Ports:     node.Endpoints{P2P: 30011, Etcd: 30012, HTTP: 40011, WS: 40012, Auth: 40013},
	}
	args, err := upgrade.LaunchArgs(n, "/data/node1", []string{"--mine"})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(args, " ")

	// the two handoff-critical flags must be present with the node's own values.
	for _, want := range []string{
		"--networkid 8285",     // uniform id so go-wemix and go-wbft peer
		"--authrpc.port 40013", // pinned per node to avoid the 8551 collision
		"--port 30011",
		"--http.port 40011",
		"--ws.port 40012",
		"--datadir /data/node1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("launch args missing %q:\n%s", want, got)
		}
	}
	// family flags are appended.
	if !strings.HasSuffix(got, "--mine") {
		t.Errorf("family flags not appended: %s", got)
	}
}
