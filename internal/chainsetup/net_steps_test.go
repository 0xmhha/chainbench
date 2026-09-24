package chainsetup_test

import (
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"

	"context"
	"github.com/0xmhha/chainbench/internal/resource"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// presetDir is the repository's shipped key set, used as a realistic fixture
// (same convention as the engine tests).
const presetDir = "../../presets/keys"

// TestChainStepPipeline composes a network step by step without a chain binary:
// new -> allocate -> keys -> genesis -> config -> launchopts -> filestore.
// It pins that each step persists its state, that the argv comes from the
// single assembly site with overrides applied, and that the lifecycle steps
// fail with actionable errors when their prerequisites are missing.
func TestChainStepPipeline(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := verb.ChainNew(ctx, d, verb.ChainNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, BPCount: 2, ENCount: 1}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if out, err := verb.ChainKeys(ctx, d, verb.ChainKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	} else if !strings.Contains(out.Detail, "identities") {
		t.Fatalf("keys detail = %q", out.Detail)
	}
	if _, err := verb.ChainGenesis(ctx, d, chainsetup.ChainGenesisIn{DataDir: dir, ChainID: 9999}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "genesis.json")); err != nil {
		t.Fatalf("genesis file: %v", err)
	} else if !strings.Contains(string(b), "9999") {
		t.Fatal("genesis does not carry the chain-id override")
	}
	if _, err := verb.ChainConfig(ctx, d, verb.ChainConfigIn{DataDir: dir}); err != nil {
		t.Fatalf("config: %v", err)
	}

	lo, err := verb.ChainLaunchOpts(ctx, d, verb.ChainLaunchOptsIn{DataDir: dir, Set: []string{"networkid=4242"}})
	if err != nil {
		t.Fatalf("launchopts: %v", err)
	}
	if len(lo.Nodes) != 3 {
		t.Fatalf("node table = %d, want 3", len(lo.Nodes))
	}
	argv := strings.Join(lo.Nodes[0].Args, " ")
	for _, frag := range []string{"--datadir", "--networkid 4242", "--unlock"} {
		if !strings.Contains(argv, frag) {
			t.Fatalf("node1 argv missing %q:\n%s", frag, argv)
		}
	}
	// node3 is an endpoint: no unlock.
	if strings.Contains(strings.Join(lo.Nodes[2].Args, " "), "--unlock") {
		t.Fatal("endpoint node must not unlock an account")
	}

	if _, err := verb.ChainProvision(ctx, d, verb.ChainProvisionIn{DataDir: dir}); err != nil {
		t.Fatalf("provision: %v", err)
	}

	// chainsetup.State round-trips: a fresh open sees the accumulated composition.
	st, err := verb.ChainStatus(ctx, d, verb.ChainStatusIn{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []string{"new", "place", "keys", "genesis", "config", "build", "deploy"} {
		if s, ok := st.State.Steps[step]; !ok || !s.Done {
			t.Fatalf("step %q not recorded: %+v", step, st.State.Steps)
		}
	}
	if len(st.State.Nodes) != 3 || len(st.State.Nodes[0].Args) == 0 {
		t.Fatalf("persisted node table incomplete: %+v", st.State.Nodes)
	}

	// Lifecycle prerequisites fail with actionable messages. Start now refuses
	// before it looks at the binary, because init has not run: launching a node
	// over a datadir no genesis ever reached is the worse of the two, and the
	// order the message arrives in is the order the operator has to fix them.
	if _, err := verb.ChainStart(ctx, d, verb.ChainStartIn{DataDir: dir}); err == nil ||
		!strings.Contains(err.Error(), "init has not run") {
		t.Fatalf("start before init must say so, got %v", err)
	}
	// init is reached (its own prerequisite, deploy, has run) and fails on the
	// binary itself, which is what the operator has to supply.
	if _, err := verb.ChainInit(ctx, d, verb.ChainInitIn{DataDir: dir}); err == nil ||
		!strings.Contains(err.Error(), "gstable") {
		t.Fatalf("init must fail naming the binary it could not run, got %v", err)
	}
	if _, err := verb.ChainRestart(ctx, d, verb.ChainRestartIn{DataDir: dir, Node: 9}); err == nil {
		t.Fatal("restart of an unknown node must fail")
	}

	// Rm clears the composed data plane and the node table.
	if _, err := verb.ChainRm(ctx, d, verb.ChainRmIn{DataDir: dir}); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "genesis.json")); !os.IsNotExist(err) {
		t.Fatal("rm must remove the genesis")
	}
	st, err = verb.ChainStatus(ctx, d, verb.ChainStatusIn{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.State.Nodes) != 0 {
		t.Fatalf("rm must clear the node table, got %+v", st.State.Nodes)
	}
}

// TestChainStepPrerequisites pins the fail-fast messages steps give before their
// prerequisites ran.
func TestChainStepPrerequisites(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}

	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, BPCount: 1}); err == nil ||
		!strings.Contains(err.Error(), "chain new") {
		t.Fatalf("allocate before new: %v", err)
	}
	if _, err := verb.ChainNew(ctx, d, verb.ChainNewIn{DataDir: dir, Chain: "stablenet"}); err != nil {
		t.Fatal(err)
	}
	if _, err := verb.ChainGenesis(ctx, d, chainsetup.ChainGenesisIn{DataDir: dir}); err == nil ||
		!strings.Contains(err.Error(), "chain place") {
		t.Fatalf("genesis before allocate: %v", err)
	}
	if _, err := verb.ChainKeys(ctx, d, verb.ChainKeysIn{DataDir: dir}); err == nil {
		t.Fatal("keys with no node count must fail")
	}
	if _, err := verb.ChainHealth(ctx, d, verb.ChainHealthIn{DataDir: dir}); err == nil {
		t.Fatal("health before allocate must fail")
	}
}

// TestChainAllocate_ProxiedKeepsEndpointsAwayFromProducers composes the graph a
// production network runs and checks it where it actually takes effect: the
// config each node is handed. Transactions arrive at an endpoint and travel
// inward, so an endpoint must not hold a route to a producer.
func TestChainAllocate_ProxiedKeepsEndpointsAwayFromProducers(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	topo := filepath.Join(dir, "topology.yaml")
	if err := os.WriteFile(topo, []byte(`chain: stablenet
nodes:
  - {index: 1, role: bp}
  - {index: 2, role: bp}
  - {index: 3, role: pn}
  - {index: 4, role: en}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := verb.ChainNew(ctx, d, verb.ChainNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, TopologyPath: topo, Peering: "proxied"}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if _, err := verb.ChainKeys(ctx, d, verb.ChainKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if _, err := verb.ChainGenesis(ctx, d, chainsetup.ChainGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if _, err := verb.ChainConfig(ctx, d, verb.ChainConfigIn{DataDir: dir}); err != nil {
		t.Fatalf("config: %v", err)
	}

	// node3 is the proxy tier: it is the only node the others may dial, and the
	// only one that sees both sides.
	pn := readConfig(t, dir, 3)
	for _, n := range []int{1, 2, 4} {
		if !strings.Contains(pn, portOf(t, dir, n)) {
			t.Fatalf("pn config should list node%d:\n%s", n, pn)
		}
	}
	en := readConfig(t, dir, 4)
	for _, bp := range []int{1, 2} {
		if strings.Contains(en, portOf(t, dir, bp)) {
			t.Fatalf("endpoint lists producer node%d — proxied means it must not:\n%s", bp, en)
		}
	}
	if !strings.Contains(en, portOf(t, dir, 3)) {
		t.Fatalf("endpoint should dial the proxy tier:\n%s", en)
	}

	// An impossible graph is refused where the layout is chosen, not later.
	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, BPCount: 2, Peering: "starfish"}); err == nil {
		t.Fatal("an unknown peering must be refused")
	}
}

// readConfig returns one node's rendered config from the workspace.
func readConfig(t *testing.T, dir string, index int) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "config_node"+itoa(index)+".toml"))
	if err != nil {
		t.Fatalf("read node%d config: %v", index, err)
	}
	return string(b)
}

// portOf returns the "@host:p2p" fragment identifying a node inside an enode.
func portOf(t *testing.T, dir string, index int) string {
	t.Helper()
	out, err := verb.ChainStatus(context.Background(), chainsetup.Deps{}, verb.ChainStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, n := range out.State.Nodes {
		if n.Index == index {
			return ":" + itoa(n.P2P) + "?"
		}
	}
	t.Fatalf("node%d not in the workspace", index)
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// TestChainAllocate_RecordsTheEtcdPort: the composed network must be able to say
// which port a node's etcd is on. It is derived by the binary (p2p+1) and is
// the reason the port plan reserves a step of two, but the workspace used to
// drop it — so the rule protected a value nothing could read back.
func TestChainAllocate_RecordsTheEtcdPort(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	// wemix, deliberately: it is the family that derives etcd ports; a wbft
	// node listens on p2p alone and records none.
	if _, err := verb.ChainNew(ctx, d, verb.ChainNewIn{DataDir: dir, Chain: "wemix", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, BPCount: 2}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	out, err := verb.ChainStatus(ctx, d, verb.ChainStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, n := range out.State.Nodes {
		if n.Etcd != n.P2P+1 {
			t.Fatalf("node%d: etcd = %d, want p2p+1 (%d)", n.Index, n.Etcd, n.P2P+1)
		}
	}
}

// TestAllocate_AllServersRecordsEachNodesServer pins the server set contract every later
// step depends on: a spread placement names each node's server-set entry, so
// files, init, and launch reach THAT machine — not the first one's.
func TestAllocate_AllServersRecordsEachNodesServer(t *testing.T) {
	dir := t.TempDir()
	set := filepath.Join(t.TempDir(), "server-set.yaml")
	if err := os.WriteFile(set, []byte(
		"version: 2\n"+
			"pool:\n"+
			"  hosts: [{name: box1, addr: 192.0.2.11}, {name: box2, addr: 192.0.2.12}, {name: box3, addr: 192.0.2.13}]\n"+
			"  slots: 2\n"+
			"ssh: {user: dev, password: pw}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := chainsetup.Deps{Clock: fixedClock()}
	if _, err := verb.ChainNew(context.Background(), d, verb.ChainNewIn{
		DataDir: dir, Chain: "stablenet", KeysDir: presetDir,
		Target: resource.Spec{DataRoot: "/data/cb"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := verb.ChainAllocate(context.Background(), d, chainsetup.ChainAllocateIn{
		DataDir: dir, BPCount: 3,
		Server: resource.ServerRef{SetPath: set, All: true},
	}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	st := stateOf(t, dir, d)
	want := []string{"box1", "box2", "box3"}
	for i, ns := range st.Nodes {
		if ns.Server != want[i] {
			t.Errorf("node%d server = %q, want %q", ns.Index, ns.Server, want[i])
		}
		if ns.Host == "" || ns.Host == st.Nodes[(i+1)%3].Host {
			t.Errorf("node%d host = %q — nodes must spread across machines", ns.Index, ns.Host)
		}
	}
}
