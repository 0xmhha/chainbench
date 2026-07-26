package upgrade_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/consensus/upgrade"
	"github.com/0xmhha/chainbench/pkg/core/portplan"
)

func TestLaunchArgs(t *testing.T) {
	n := upgrade.NodeSpec{
		NetworkID: 8285,
		Ports:     portplan.Ports{P2P: 30011, Etcd: 30012, HTTP: 40011, WS: 40012, Auth: 40013},
	}
	got := strings.Join(upgrade.LaunchArgs(n, "/data/node1", "/data/config.toml", []string{"--mine"}), " ")

	// the two handoff-critical flags must be present with the node's own values.
	for _, want := range []string{
		"--networkid 8285",     // uniform id so go-wemix and go-wbft peer
		"--authrpc.port 40013", // pinned per node to avoid the 8551 collision
		"--port 30011",
		"--http.port 40011",
		"--ws.port 40012",
		"--datadir /data/node1",
		"--config /data/config.toml",
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
