package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/target"
)

// Ports and host addresses are site-specific and must come from the operator's
// gitignored inventory, with the same entry shape for this machine and a remote
// host. These pin that the server set actually decides the layout, and that the
// no-inventory path stays usable and says where its ports came from.

// writeInventory writes a server set and returns its path.
func writeInventory(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "server-set.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write inventory: %v", err)
	}
	return p
}

const localInventory = `
version: 2
pool:
  hosts: [{name: local, addr: 127.0.0.1}]
  slots: 8
  ports: {p2p: {base: 30303, step: 10}, rpc: {base: 8545, step: 10}}
`

func TestNetAllocate_InventoryDecidesThePorts(t *testing.T) {
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, _ := filepath.Abs(presetDir)
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}

	out, err := app.NetAllocate(ctx, d, app.NetAllocateIn{
		DataDir: dir, Validators: 2,
		Server: app.ServerRef{SetPath: writeInventory(t, localInventory), Name: "local"},
	})
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	st := stateOf(t, dir, d)
	if st.Nodes[0].HTTP != 8545 || st.Nodes[1].HTTP != 8555 {
		t.Errorf("http ports = %d, %d; want the server set's 8545/8555", st.Nodes[0].HTTP, st.Nodes[1].HTTP)
	}
	if st.Nodes[0].P2P != 30303 {
		t.Errorf("p2p = %d, want the server set's 30303", st.Nodes[0].P2P)
	}
	// Where the plan came from is recorded, not left for an operator to guess.
	if !strings.Contains(st.PortSource, "local") || !strings.Contains(out.Detail, "ports:") {
		t.Errorf("port source = %q, detail = %q", st.PortSource, out.Detail)
	}
}

func TestNetAllocate_WithoutAnInventoryUsesTheBuiltinsAndSaysSo(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 2})
	st := stateOf(t, dir, d)
	if !strings.Contains(st.PortSource, "built-in") {
		t.Errorf("port source = %q, want it to name the built-ins", st.PortSource)
	}
	if st.Nodes[0].HTTP == 0 {
		t.Error("no ports assigned without a server set")
	}
}

func TestNetAllocate_RemoteServerRetargetsTheDataPlane(t *testing.T) {
	// The same entry shape describes a remote host: only kind and ssh differ,
	// and the composition follows the target it names.
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, _ := filepath.Abs(presetDir)
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	inv := writeInventory(t, `
version: 2
pool:
  hosts: [{name: bp1, addr: 10.0.0.11}]
  ports: {p2p: {base: 30303, step: 10}, rpc: {base: 8545, step: 10}}
ssh: {user: deploy, port: 2222}
dataRoot: /srv/chainbench
`)

	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{
		DataDir: dir, Validators: 1,
		Server: app.ServerRef{SetPath: inv, Name: "bp1"},
	}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	st := stateOf(t, dir, d)
	// The target names the set entry rather than flattening its login: the
	// server set stays the single credential source for every later step.
	if st.Target.Kind != target.TargetServer || st.Target.Server != "bp1" {
		t.Fatalf("target = %+v, want a server-set target naming bp1", st.Target)
	}
	if st.Target.Host != "10.0.0.11" {
		t.Errorf("host not carried for addressing: %+v", st.Target)
	}
	if st.Target.User != "" || st.Target.Port != 0 {
		t.Errorf("login fields must stay in the server set, not the spec: %+v", st.Target)
	}
	if st.Target.DataRoot != "/srv/chainbench" {
		t.Errorf("data root = %q, want the server set's", st.Target.DataRoot)
	}
	// The node's own address is recorded, so a NodeSet reader reaches the host
	// rather than this machine.
	ns, err := app.NetworkStatus(ctx, d, app.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(ns.Nodes.Nodes[0].RPCURL, "10.0.0.11") {
		t.Errorf("rpc url = %q, want the remote host", ns.Nodes.Nodes[0].RPCURL)
	}
}

func TestNetAllocate_FleetSpreadsOneNodePerHost(t *testing.T) {
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, _ := filepath.Abs(presetDir)
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	inv := writeInventory(t, `
version: 2
pool:
  hosts: [{name: bp1, addr: 10.0.0.11}, {name: bp2, addr: 10.0.0.12}, {name: bp3, addr: 10.0.0.13}]
  slots: 1
  ports: {p2p: {base: 30303, step: 10}, rpc: {base: 8545, step: 10}}
ssh: {user: deploy}
dataRoot: /srv/cb
`)

	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{
		DataDir: dir, Validators: 3,
		Server: app.ServerRef{SetPath: inv, Fleet: true},
	}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	// One node per host, every host on the same ports (requirement 16).
	st := stateOf(t, dir, d)
	hosts := map[string]bool{}
	for _, n := range st.Nodes {
		hosts[n.Host] = true
		if n.HTTP != 8545 {
			t.Errorf("node%d http = %d, want the same 8545 on every host", n.Index, n.HTTP)
		}
	}
	if len(hosts) != 3 {
		t.Errorf("nodes landed on %d hosts, want 3: %v", len(hosts), hosts)
	}
}

func TestResolveServer_NamingAServerWithoutAnInventoryIsAnError(t *testing.T) {
	// Silently falling back to built-in ports when the operator asked for a
	// named server would put nodes somewhere they did not choose.
	_, err := app.ResolveServer(app.Deps{}, app.ServerRef{Name: "bp1"}, 1, 100)
	if err == nil {
		t.Fatal("want an error naming a server with no server set")
	}
	if !strings.Contains(err.Error(), "sample") {
		t.Errorf("error should point at the sample, got: %v", err)
	}
}

func TestResolveServer_NoSelectionFallsBackToTheBuiltins(t *testing.T) {
	out, err := app.ResolveServer(app.Deps{}, app.ServerRef{}, 1, 100)
	if err != nil {
		t.Fatalf("ResolveServer: %v", err)
	}
	if out.HasTarget {
		t.Error("no server set should leave the workspace target alone")
	}
	if !strings.Contains(out.Placement.Source, "built-in") {
		t.Errorf("source = %q", out.Placement.Source)
	}
}

func TestNetAllocate_FleetRecordsEachNodesOwnHost(t *testing.T) {
	// The per-node host is what the config's static-node list and the launch
	// specs derive from. If every node recorded this machine instead, a fleet's
	// nodes could not find their peers and the failure would look like a
	// consensus problem rather than an addressing one.
	//
	// Writing a fleet's configs needs a reachable SSH host, so this asserts the
	// node table the writers read rather than the written files.
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, _ := filepath.Abs(presetDir)
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	inv := writeInventory(t, `
version: 2
pool:
  hosts: [{name: bp1, addr: 10.0.0.11}, {name: bp2, addr: 10.0.0.12}, {name: bp3, addr: 10.0.0.13}]
  slots: 1
  ports: {p2p: {base: 30303, step: 10}, rpc: {base: 8545, step: 10}}
ssh: {user: deploy}
dataRoot: /srv/cb
`)
	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{
		DataDir: dir, Validators: 2,
		Server: app.ServerRef{SetPath: inv, Fleet: true},
	}); err != nil {
		t.Fatalf("allocate: %v", err)
	}

	nodes := stateOf(t, dir, d).Nodes
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].Host != "10.0.0.11" || nodes[1].Host != "10.0.0.12" {
		t.Errorf("node hosts = %q, %q; want the server set's", nodes[0].Host, nodes[1].Host)
	}
	for _, n := range nodes {
		if n.Host == "127.0.0.1" {
			t.Errorf("node%d recorded this machine instead of its server", n.Index)
		}
	}
}
