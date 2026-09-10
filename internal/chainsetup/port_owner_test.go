package chainsetup

import (
	"os"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestRecordedLeftovers_KeepsTheServerWithThePort is the sibling of the
// running-node collision (MON-010), found by running two compositions against
// one docker server.
//
// Spread across a server set, every server runs its slot-1 node on the same
// numbers: node1 on server6 and node2 on server7 both plan 8601. A map keyed on
// the number alone answers "who planned 8601?" with whichever node was written
// last, so a busy port on one server is reported as a node on another and the
// operator goes looking on the wrong machine.
func TestRecordedLeftovers_KeepsTheServerWithThePort(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	// Two nodes on two servers, planning the identical port numbers — which is
	// what one node per host produces, not a contrived case.
	w.state.Nodes = []node.Record{
		{Index: 1, Label: "node1", Host: "172.30.0.16", Endpoints: node.Endpoints{HTTP: 8601, P2P: 30301}, PID: 111},
		{Index: 2, Label: "node2", Host: "172.30.0.17", Endpoints: node.Endpoints{HTTP: 8601, P2P: 30301}, PID: 222},
	}

	mine := w.recordedLeftovers()

	got, ok := mine[portKey{host: "172.30.0.16", port: 8601}]
	if !ok {
		t.Fatal("server6's 8601 was not recorded at all")
	}
	if got.node != 1 || got.pid != 111 {
		t.Fatalf("172.30.0.16:8601 = node%d pid %d, want node1 pid 111 — the port was attributed to the other server's node", got.node, got.pid)
	}
	got, ok = mine[portKey{host: "172.30.0.17", port: 8601}]
	if !ok {
		t.Fatal("server7's 8601 was not recorded at all")
	}
	if got.node != 2 || got.pid != 222 {
		t.Fatalf("172.30.0.17:8601 = node%d pid %d, want node2 pid 222", got.node, got.pid)
	}

	// An address no node planned belongs to nobody here, however familiar the
	// port number looks.
	if _, ok := mine[portKey{host: "172.30.0.99", port: 8601}]; ok {
		t.Fatal("a port on an unplanned host was claimed by this workspace")
	}
}
