package app

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func spreadSet(t *testing.T, hosts string) *resource.Set {
	t.Helper()
	set, err := deploymentSet(DeploymentDocumentInput{Kind: "server-set", Content: json.RawMessage(`{"version":2,"pool":{"hosts":[` + hosts + `],"slots":2,"ports":{"p2p":{"base":31000,"step":10},"rpc":{"base":8600,"step":10}}}}`)})
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestSpreadPlanResolvesEveryServerOfTheSet(t *testing.T) {
	set := spreadSet(t, `{"name":"a","addr":"10.0.0.1"},{"name":"b","addr":"10.0.0.2"}`)
	servers, err := webPlanServers(set, webChainArguments{AllServers: true}, "chain.deploy")
	if err != nil || len(servers) != 2 || servers[0].Name != "a" || servers[1].Host != "10.0.0.2" {
		t.Fatalf("%+v %v", servers, err)
	}
	one, err := webPlanServers(set, webChainArguments{ServerRef: "b"}, "node.stop")
	if err != nil || len(one) != 1 || one[0].Name != "b" {
		t.Fatalf("%+v %v", one, err)
	}
	for _, c := range []struct {
		args webChainArguments
		op   string
	}{
		{webChainArguments{AllServers: true, ServerRef: "a"}, "chain.deploy"},
		{webChainArguments{AllServers: true}, "test.run"},
	} {
		if _, err := webPlanServers(set, c.args, c.op); err == nil {
			t.Errorf("accepted %+v for %s", c.args, c.op)
		}
	}
	// A recorded spread network is controlled and collected over the whole set.
	for _, op := range []string{"node.stop", "node.swap", "test.attach", "network.monitor"} {
		if all, err := webPlanServers(set, webChainArguments{AllServers: true}, op); err != nil || len(all) != 2 {
			t.Errorf("%s over the whole set: %v", op, err)
		}
	}
	if _, err := webPlanServers(spreadSet(t, `{"name":"a","addr":"10.0.0.1"}`), webChainArguments{AllServers: true}, "chain.deploy"); err == nil {
		t.Error("a one-server set was spread")
	}
}

func TestSpreadClaimsGiveEachMachineOnlyItsOwnPorts(t *testing.T) {
	set := spreadSet(t, `{"name":"a","addr":"10.0.0.1"},{"name":"b","addr":"10.0.0.2"}`)
	pool, err := set.Pool(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	placement, err := resource.Assign(pool, []resource.Request{{Role: node.RoleBP}, {Role: node.RoleBP}, {Role: node.RoleBP}})
	if err != nil {
		t.Fatal(err)
	}
	servers, _ := webPlanServers(set, webChainArguments{AllServers: true}, "chain.deploy")
	targets := []resource.Inspection{{HostIdentity: "host-a", DataPath: "/data"}, {HostIdentity: "host-b", DataPath: "/data"}}
	claims := webTargetClaims(servers, targets, placement)
	if len(claims) != 2 || claims[0].HostIdentity != "host-a" || claims[1].HostIdentity != "host-b" {
		t.Fatalf("%+v", claims)
	}
	onB := map[int]bool{}
	for _, entry := range placement.Placements() {
		if entry.Host == "10.0.0.2" {
			onB[entry.Ports.P2P] = true
		}
	}
	if len(onB) == 0 {
		t.Fatal("no node was placed on the second machine")
	}
	for _, p := range claims[0].Ports {
		if onB[p] && !portOnHost(placement, "10.0.0.1", p) {
			t.Errorf("machine a claims port %d of a node on machine b", p)
		}
	}
	for p := range onB {
		found := false
		for _, q := range claims[1].Ports {
			found = found || q == p
		}
		if !found {
			t.Errorf("machine b does not claim its node's port %d", p)
		}
	}
}

func portOnHost(m *node.Map, host string, port int) bool {
	for _, entry := range m.Placements() {
		if entry.Host == host && entry.Ports.P2P == port {
			return true
		}
	}
	return false
}

func TestSpreadRecordsAreDetected(t *testing.T) {
	one := State{Nodes: []node.Record{{Index: 1, Server: "a", Host: "10.0.0.1"}, {Index: 2, Server: "a", Host: "10.0.0.1"}}}
	two := State{Nodes: []node.Record{{Index: 1, Server: "a", Host: "10.0.0.1"}, {Index: 2, Server: "b", Host: "10.0.0.2"}}}
	if webRecordSpansServers(one) || !webRecordSpansServers(two) {
		t.Fatal("spread detection")
	}
}

func TestSpreadPayloadTargetsListEveryMachine(t *testing.T) {
	single := webChainPayload{Target: resource.Inspection{HostIdentity: "a"}}
	if got := webPayloadTargets(single); len(got) != 1 || got[0].HostIdentity != "a" {
		t.Fatalf("%+v", got)
	}
	spread := webChainPayload{Target: resource.Inspection{HostIdentity: "a"}, Targets: []resource.Inspection{{HostIdentity: "a"}, {HostIdentity: "b"}}}
	if got := webPayloadTargets(spread); len(got) != 2 || got[1].HostIdentity != "b" {
		t.Fatalf("%+v", got)
	}
}

func TestSpreadNodesResolveToTheirOwnServerMachine(t *testing.T) {
	p := webChainPayload{Target: resource.Inspection{HostIdentity: "a"}, Targets: []resource.Inspection{{HostIdentity: "a"}, {HostIdentity: "b"}}, TargetServers: []string{"one", "two"}}
	if m, ok := webNodeMachine(p, node.Record{Server: "two"}); !ok || m.HostIdentity != "b" {
		t.Fatalf("%+v %v", m, ok)
	}
	if _, ok := webNodeMachine(p, node.Record{Server: "three"}); ok {
		t.Fatal("a node on an unpinned server resolved")
	}
	single := webChainPayload{Target: resource.Inspection{HostIdentity: "only"}}
	if m, ok := webNodeMachine(single, node.Record{}); !ok || m.HostIdentity != "only" {
		t.Fatalf("%+v %v", m, ok)
	}
	if got := webRecordedMachines(p); len(got) != 2 || got[1].Server != "two" || got[1].Target.HostIdentity != "b" {
		t.Fatalf("%+v", got)
	}
}

func TestSpreadRecordedPlacementAcceptsEachServerOfTheSet(t *testing.T) {
	set := spreadSet(t, `{"name":"a","addr":"10.0.0.1"},{"name":"b","addr":"10.0.0.2"}`)
	pool, err := set.Pool(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	ports, err := resource.PlanBands(1, pool.Ports, pool.Reservation)
	if err != nil {
		t.Fatal(err)
	}
	state := State{Nodes: []node.Record{{Index: 1, Server: "a", Host: "10.0.0.1"}, {Index: 2, Server: "b", Host: "10.0.0.2"}}}
	for i := range state.Nodes {
		state.Nodes[i].Endpoints = ports
	}
	hosts := map[string]bool{}
	for _, h := range pool.Hosts {
		hosts[h.Addr] = true
	}
	for _, ns := range state.Nodes {
		if !hosts[ns.Host] {
			t.Fatalf("%s is not a host of the whole-set pool", ns.Host)
		}
	}
	if !webRecordSpansServers(state) {
		t.Fatal("the record is spread")
	}
}

// A spread plan visits every server of the set, but writes only to those that
// get a node. Four nodes on a fifteen-server pool must leave the other eleven
// free for the next job.
func TestSpreadClaimsLeaveServersWithoutNodesFree(t *testing.T) {
	servers := []resource.Server{{Name: "a", Host: "192.0.2.1"}, {Name: "b", Host: "192.0.2.2"}}
	pool := resource.Pool{
		Hosts: []resource.Host{{Name: "a", Addr: "192.0.2.1"}, {Name: "b", Addr: "192.0.2.2"}},
		Slots: 4,
		Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
	}
	placement, err := resource.Assign(pool, []resource.Request{{Role: node.RoleBP, Server: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	targets := []resource.Inspection{{HostIdentity: "host-a", DataPath: "/data"}, {HostIdentity: "host-b", DataPath: "/data"}}
	claims := webTargetClaims(servers, targets, placement)
	if claims[0].Idle || !claims[1].Idle {
		t.Fatalf("claims = %+v, want only the empty server idle", claims)
	}
	next := WebResourceClaim{HostIdentity: "host-b", DataPath: "/data", Ports: []int{31000}}
	if claimsOverlap(claims[1], next) || claimsOverlap(next, claims[1]) {
		t.Fatal("a server without nodes blocked the next job")
	}
	occupied := WebResourceClaim{HostIdentity: "host-a", DataPath: "/data"}
	if !claimsOverlap(claims[0], occupied) {
		t.Fatal("a server holding a node stopped conflicting")
	}
}

// A workspace records one network. Planning elsewhere used to read the recorded
// server without its login and answer a bare 403.
func TestRecordedTargetOutsideTheSelectionNamesTheServer(t *testing.T) {
	servers := []resource.Server{{Name: "server5"}}
	err := webRecordOnSelectedServers(resource.Spec{Server: "server1", DataRoot: "/data"}, servers)
	if err == nil || !strings.Contains(err.Error(), "server1") {
		t.Fatalf("err = %v, want the recorded server named", err)
	}
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("err = %v, want a conflict", err)
	}
	if err = webRecordOnSelectedServers(resource.Spec{Server: "server5", DataRoot: "/data"}, servers); err != nil {
		t.Fatal(err)
	}
	if err = webRecordOnSelectedServers(resource.Spec{DataRoot: "/data"}, servers); err != nil {
		t.Fatalf("a local record needs no login: %v", err)
	}
}

// A composition that failed before launch recorded its nodes without a launch.
// Nothing can be running for them, so a retry must not be refused.
func TestNodeNeverLaunchedHasNothingToVerify(t *testing.T) {
	if !webNodeNeverLaunched(node.Record{}) {
		t.Fatal("a node without a launch was checked for a process")
	}
	if webNodeNeverLaunched(node.Record{Args: []string{"--datadir", "/data"}}) {
		t.Fatal("a stopped node with a recorded launch skipped the check")
	}
	if webNodeNeverLaunched(node.Record{PID: 7}) {
		t.Fatal("a node with a PID skipped the check")
	}
}

// wemix names its IPC socket after the executable, so a node run as "payload"
// listens where the bootstrap never looks.
func TestRemoteBinaryKeepsTheManifestsName(t *testing.T) {
	if got := webRemoteBinaryName("gwemix", "/data/assets/abc/payload"); got != "gwemix" {
		t.Fatalf("name = %q", got)
	}
	if got := webRemoteBinaryName("", "/data/assets/abc/payload"); got != "payload" {
		t.Fatalf("a manifest without a binary name gave %q", got)
	}
}
