package poa_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// fakeChain answers the console calls the join makes, and records them. It
// stands in for four nodes: which one is being asked is the IPC path in the
// call, exactly as the real thing distinguishes them.
type fakeChain struct {
	mu sync.Mutex
	// calls are the --exec expressions, prefixed by the node they went to.
	calls []string
	// cluster is what the boot node reports; joins add to it.
	cluster []string
	// stubborn is a node whose first join does nothing, which is the behaviour
	// that made a four-producer network come up with three in the cluster.
	stubborn string
	asked    map[string]int
}

func (f *fakeChain) run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ipc, exec string
	for i, a := range args {
		if a == "attach" && i+1 < len(args) {
			ipc = args[i+1]
		}
		if a == "--exec" && i+1 < len(args) {
			exec = args[i+1]
		}
	}
	who := filepath.Base(filepath.Dir(ipc))
	f.calls = append(f.calls, who+": "+exec)

	switch {
	case strings.Contains(exec, "admin.wemixInfo.nodes"):
		return []byte(`"[{\"name\":\"node1\"},{\"name\":\"node2\"}]"`), nil
	case strings.Contains(exec, "admin.etcdJoin"):
		if f.asked == nil {
			f.asked = map[string]int{}
		}
		f.asked[who]++
		if who == f.stubborn && f.asked[who] == 1 {
			return []byte("null"), nil // returns fine, joins nothing
		}
		f.cluster = append(f.cluster, who)
		return []byte("null"), nil
	case strings.Contains(exec, "admin.wemixInfo.etcd.cluster"):
		parts := make([]string, 0, len(f.cluster))
		for _, m := range f.cluster {
			parts = append(parts, m+"=https://127.0.0.1:31001")
		}
		return []byte(`"` + strings.Join(parts, ",") + `"`), nil
	}
	return []byte("null"), nil
}

func (f *fakeChain) execs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// planWithIPCs builds a plan whose datadirs hold a live-looking IPC socket, so
// the action's wait for the socket is satisfied without a chain.
func planWithIPCs(t *testing.T, roles []node.Role) driver.Plan {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "cbj")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	specs := make([]driver.NodeSpec, 0, len(roles))
	for i, r := range roles {
		dir := filepath.Join(root, fmt.Sprintf("node%d", i+1))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		listenUnix(t, filepath.Join(dir, "gwemix.ipc"))
		specs = append(specs, driver.NodeSpec{Index: i + 1, Role: r, DataDir: dir, Binary: "gwemix"})
	}
	return driver.Plan{DataRoot: root, Nodes: specs}
}

// listenUnix creates a real unix socket at path. The wait before an action
// checks the file's mode, so a plain file would not do.
func listenUnix(t *testing.T, path string) {
	t.Helper()
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
}

// TestEtcdJoin_AsksTheBootNodeAndOnlyProducers pins the two things that were
// wrong when a four-producer network sealed every block on one node.
//
// The join is directed at the peer that has the cluster, not at the node doing
// the joining — calling it with the joiner's own name returns without error and
// joins nothing, which is how the mistake hides. And it is a producers-only
// handshake: the chain answers it for governance members, so asking an endpoint
// to join is asking for a refusal.
func TestEtcdJoin_AsksTheBootNodeAndOnlyProducers(t *testing.T) {
	plan := planWithIPCs(t, []node.Role{node.RoleBP, node.RoleBP, node.RoleBP, node.RoleEN})
	chain := &fakeChain{cluster: []string{"node1"}}
	b := poa.Bootstrap{Binary: "gwemix", Run: chain.run}

	boot := node.Node{Index: 1, Role: node.RoleBP}
	if err := b.Action(context.Background(), poa.ActionEtcdJoin, plan, boot); err != nil {
		t.Fatalf("etcd-join: %v", err)
	}

	var joins []string
	for _, c := range chain.execs() {
		if strings.Contains(c, "admin.etcdJoin") {
			joins = append(joins, c)
		}
	}
	if len(joins) != 2 {
		t.Fatalf("joins = %v, want one per non-boot producer", joins)
	}
	for _, j := range joins {
		if !strings.Contains(j, `admin.etcdJoin("node1")`) {
			t.Fatalf("join asked the wrong peer: %s — the argument is the node that has the cluster", j)
		}
		if strings.HasPrefix(j, "node4:") {
			t.Fatalf("the endpoint was asked to join: %s", j)
		}
	}
}

// TestEtcdJoin_KeepsAskingUntilTheClusterSaysSo: a join that returns null has
// not necessarily happened. Measured live, one producer of four answered
// cleanly and stayed outside the cluster until it was asked again, so the
// cluster is the evidence rather than the return value.
func TestEtcdJoin_KeepsAskingUntilTheClusterSaysSo(t *testing.T) {
	plan := planWithIPCs(t, []node.Role{node.RoleBP, node.RoleBP})
	chain := &fakeChain{cluster: []string{"node1"}, stubborn: "node2"}
	b := poa.Bootstrap{Binary: "gwemix", Run: chain.run}

	if err := b.Action(context.Background(), poa.ActionEtcdJoin, plan, node.Node{Index: 1, Role: node.RoleBP}); err != nil {
		t.Fatalf("etcd-join: %v", err)
	}
	if chain.asked["node2"] < 2 {
		t.Fatalf("node2 was asked %d time(s); a silent no-op join must be retried", chain.asked["node2"])
	}
}

// TestEtcdJoin_LoneProducerJoinsNothing: with one producer the cluster of one
// is already formed, and there is nobody to ask.
func TestEtcdJoin_LoneProducerJoinsNothing(t *testing.T) {
	plan := planWithIPCs(t, []node.Role{node.RoleBP, node.RoleEN})
	chain := &fakeChain{cluster: []string{"node1"}}
	b := poa.Bootstrap{Binary: "gwemix", Run: chain.run}

	if err := b.Action(context.Background(), poa.ActionEtcdJoin, plan, node.Node{Index: 1, Role: node.RoleBP}); err != nil {
		t.Fatalf("etcd-join: %v", err)
	}
	for _, c := range chain.execs() {
		if strings.Contains(c, "admin.etcdJoin") {
			t.Fatalf("a lone producer asked somebody to join: %s", c)
		}
	}
}
