package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chains, as a surface does
)

// NetPlan is the allocator asked as a question: "where would these nodes land".
// It was uncovered, and what it answers is load-bearing in a way a wrong answer
// does not announce -- a plan that says five servers and composes onto one is a
// shape this project has already met once, from the other side of the boundary.
//
// The allocation order is also a stated requirement: hand out slot 1 on every
// server, then wrap back to the first server at the NEXT port band. Nothing at
// the level an operator uses asserted it.

// serverSet writes a set file with n hosts and returns its path.
func serverSet(t *testing.T, n, slots int) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("version: 2\npool:\n  hosts:\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "    - { name: server%d, addr: 10.0.0.%d }\n", i, i)
	}
	fmt.Fprintf(&b, "  slots: %d\n", slots)
	// The purpose bands the real sets declare. With only p2p and rpc, the loader
	// demands rpcStep >= 3 so http/ws/auth can be derived from it; declaring the
	// bands is the shape env/docker uses.
	b.WriteString("  ports:\n    p2p:     { base: 30301, step: 3 }\n")
	b.WriteString("    rpc:     { base: 8601, step: 1 }\n    ws:      { base: 8701, step: 1 }\n")
	b.WriteString("    auth:    { base: 8501, step: 1 }\n    metrics: { base: 6060, step: 0 }\n")
	b.WriteString("ssh:\n  user: 'u'\n  insecure_host_key: true\n  password: 'p'\n")
	p := filepath.Join(t.TempDir(), "server-set.yaml")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNetPlan_RefusesANetworkWithNoValidator(t *testing.T) {
	for _, n := range []int{0, -1} {
		_, err := NetPlan(context.Background(), Deps{}, NetPlanIn{Validators: n})
		if err == nil {
			t.Errorf("%d validators was accepted", n)
			continue
		}
		if !strings.Contains(err.Error(), "nothing seals without one") {
			t.Errorf("the refusal should say why one is needed: %v", err)
		}
	}
}

// TestNetPlan_PlansWhatWasAskedFor: the plan has to be the requested shape, in
// total and per role. A plan short of what was asked for and silent about it is
// the failure worth ruling out.
func TestNetPlan_PlansWhatWasAskedFor(t *testing.T) {
	out, err := NetPlan(context.Background(), Deps{}, NetPlanIn{
		Chain: "wbft", Validators: 4, Endpoints: 2,
		Server: ServerRef{SetPath: serverSet(t, 3, 6), All: true},
	})
	if err != nil {
		t.Fatalf("NetPlan: %v", err)
	}
	if out.Total != 6 {
		t.Fatalf("planned %d nodes for 4 validators and 2 endpoints", out.Total)
	}
	if out.Roles["bp"] != 4 || out.Roles["en"] != 2 {
		t.Fatalf("roles = %v, want 4 bp and 2 en", out.Roles)
	}
	if len(out.Entries) != out.Total {
		t.Fatalf("%d entries for a total of %d", len(out.Entries), out.Total)
	}
	// A placement with no host or no port is not a placement.
	for _, e := range out.Entries {
		if e.Host == "" {
			t.Errorf("node%d was planned onto no host", e.Node)
		}
		if e.P2P == 0 || e.HTTP == 0 {
			t.Errorf("node%d was planned with no ports: %+v", e.Node, e.Endpoints)
		}
	}
}

// TestNetPlan_FillsEveryServerBeforeReusingOne is the stated allocation order.
// Six nodes over three servers must be two per server at DIFFERENT port bands,
// not three on the first and three on the second -- the wrap is what lets a
// 15-container environment hold thirty nodes, one per purpose per machine.
func TestNetPlan_FillsEveryServerBeforeReusingOne(t *testing.T) {
	out, err := NetPlan(context.Background(), Deps{}, NetPlanIn{
		Chain: "wbft", Validators: 6,
		Server: ServerRef{SetPath: serverSet(t, 3, 6), All: true},
	})
	if err != nil {
		t.Fatalf("NetPlan: %v", err)
	}
	perHost := map[string][]int{}
	for _, e := range out.Entries {
		perHost[e.Host] = append(perHost[e.Host], e.P2P)
	}
	if len(perHost) != 3 {
		t.Fatalf("six nodes landed on %d host(s), want all 3 used before any is reused", len(perHost))
	}
	for host, ports := range perHost {
		if len(ports) != 2 {
			t.Errorf("%s holds %d nodes, want 2", host, len(ports))
		}
		// Two nodes on one machine must not share a port.
		if len(ports) == 2 && ports[0] == ports[1] {
			t.Errorf("%s placed two nodes on p2p %d — the wrap must change the band", host, ports[0])
		}
	}
	// The first three nodes take the first slot on each server, in order.
	first := out.Entries[:3]
	seen := map[string]bool{}
	for _, e := range first {
		if seen[e.Host] {
			t.Errorf("node%d reused host %s before every server had one", e.Node, e.Host)
		}
		seen[e.Host] = true
	}
}

// TestNetPlan_UnknownChainIsRefused: the family supplies the port reservation,
// so a chain nobody registered cannot be planned for.
func TestNetPlan_UnknownChainIsRefused(t *testing.T) {
	if _, err := NetPlan(context.Background(), Deps{}, NetPlanIn{Chain: "nope", Validators: 1}); err == nil {
		t.Fatal("a plan was produced for an unregistered chain")
	}
}

// TestNetPlan_TooManyNodesForTheSetIsRefused: capacity is the one thing a plan
// must not round down. Asking for more than the set holds has to fail, not
// quietly return fewer nodes than were asked for.
func TestNetPlan_TooManyNodesForTheSetIsRefused(t *testing.T) {
	out, err := NetPlan(context.Background(), Deps{}, NetPlanIn{
		Chain: "wbft", Validators: 9,
		Server: ServerRef{SetPath: serverSet(t, 2, 2), All: true}, // holds 4
	})
	if err == nil {
		t.Fatalf("nine nodes were planned onto a set holding four: total=%d", out.Total)
	}
	// And it refuses for the right reason, with the arithmetic an operator needs
	// to fix it. A refusal that merely said "cannot plan" would pass the check
	// above while telling them nothing.
	for _, want := range []string{"the set is full", "9 nodes requested", "5 short"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal should say %q: %v", want, err)
		}
	}
}
