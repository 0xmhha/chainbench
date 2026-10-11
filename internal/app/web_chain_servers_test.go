package app

import (
	"encoding/json"
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
		{webChainArguments{AllServers: true}, "node.stop"},
		{webChainArguments{AllServers: true}, "test.run"},
	} {
		if _, err := webPlanServers(set, c.args, c.op); err == nil {
			t.Errorf("accepted %+v for %s", c.args, c.op)
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
