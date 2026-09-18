package upgrade

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// listenSomewhere holds a real port for the duration of the test and says which
// one, so the check is measured against the kernel rather than against a mock
// that agrees with it by construction.
func listenSomewhere(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func handoffWithPorts(ports node.Endpoints) *Handoff {
	return &Handoff{
		Plan: Plan{Nodes: []NodeSpec{{Index: 0, Ports: ports}}},
		in:   HandoffInputs{Host: "127.0.0.1"},
	}
}

// TestCheckVacant_ABusyPortIsRefusedByName.
//
// A node launched onto a taken port exits on "address already in use" and the
// handoff then waits out its timeout on an IPC socket that will never appear,
// reporting neither the port nor who holds it. Measured: a leftover node held
// the producer's p2p port and three consecutive runs died as "the node's IPC
// socket never appeared within 30s".
func TestCheckVacant_ABusyPortIsRefusedByName(t *testing.T) {
	busy := listenSomewhere(t)
	h := handoffWithPorts(node.Endpoints{P2P: busy, HTTP: 0})
	err := h.checkVacant(context.Background(), nil)
	if err == nil {
		t.Fatal("a launch onto a port something else holds was allowed")
	}
	if !strings.Contains(err.Error(), strconv.Itoa(busy)) {
		t.Fatalf("the refusal does not name the port that is taken: %v", err)
	}
	if !strings.Contains(err.Error(), "p2p") {
		t.Fatalf("the refusal does not say what the port was for: %v", err)
	}
}

// TestCheckVacant_FreePortsPass: the check must not invent collisions, or every
// handoff refuses to start.
func TestCheckVacant_FreePortsPass(t *testing.T) {
	free := listenSomewhere(t) // taken during the call above...
	h := handoffWithPorts(node.Endpoints{P2P: free})
	if err := h.checkVacant(context.Background(), nil); err == nil {
		t.Fatal("precondition: a held port should be reported busy")
	}
	// ...and the same port, once the plan does not mention it, is not a
	// collision: an unset port is not a port.
	h2 := handoffWithPorts(node.Endpoints{})
	if err := h2.checkVacant(context.Background(), nil); err != nil {
		t.Fatalf("a plan with no ports was refused: %v", err)
	}
}

// TestCheckVacant_OnlyChecksThePhaseBeingLaunched: the producer comes up alone,
// so the ports of nodes that are not starting yet are not this launch's
// business.
func TestCheckVacant_OnlyChecksThePhaseBeingLaunched(t *testing.T) {
	busy := listenSomewhere(t)
	h := &Handoff{
		Plan: Plan{Nodes: []NodeSpec{
			{Index: 0, Ports: node.Endpoints{P2P: 0}},
			{Index: 1, Ports: node.Endpoints{P2P: busy}},
		}},
		in: HandoffInputs{Host: "127.0.0.1"},
	}
	if err := h.checkVacant(context.Background(), []int{0}); err != nil {
		t.Fatalf("launching node1 alone was refused over node2's port: %v", err)
	}
	if err := h.checkVacant(context.Background(), []int{1}); err == nil {
		t.Fatal("launching node2 onto its taken port was allowed")
	}
}
