package inspector_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/inspector"
)

// TestScan_ReportsOnlyWhatAnswers: a connection that opens is the evidence.
// A refused one means nothing is there, and neither is an error the caller
// should act on — it learns about unreachable hosts when it tries to launch.
func TestScan_ReportsOnlyWhatAnswers(t *testing.T) {
	open := map[string]bool{"10.0.0.1:8600": true, "10.0.0.2:31000": true}
	dial := func(_ context.Context, hp string) (net.Conn, error) {
		if open[hp] {
			c1, c2 := net.Pipe()
			_ = c2.Close()
			return c1, nil
		}
		return nil, errors.New("connection refused")
	}

	busy := inspector.Ports(context.Background(), []inspector.Addr{
		{Host: "10.0.0.1", Port: 8600, Node: 1, Purpose: "http"},
		{Host: "10.0.0.1", Port: 31000, Node: 1, Purpose: "p2p"},
		{Host: "10.0.0.2", Port: 31000, Node: 2, Purpose: "p2p"},
	}, dial)

	if len(busy) != 2 {
		t.Fatalf("busy = %v, want the two that answered", busy)
	}
	if busy[0].Node != 1 || busy[0].Purpose != "http" {
		t.Fatalf("busy[0] = %v; findings are ordered by node then port", busy[0])
	}
}

// TestScan_SkipsPortsAFamilyDoesNotUse: a family that does not embed etcd
// leaves those ports zero, and probing port 0 would be a question about
// nothing.
func TestScan_SkipsPortsAFamilyDoesNotUse(t *testing.T) {
	asked := 0
	dial := func(_ context.Context, _ string) (net.Conn, error) {
		asked++
		return nil, errors.New("refused")
	}
	inspector.Ports(context.Background(), []inspector.Addr{
		{Host: "h", Port: 0, Node: 1, Purpose: "etcd"},
		{Host: "h", Port: 8600, Node: 1, Purpose: "http"},
	}, dial)
	if asked != 1 {
		t.Fatalf("dialled %d time(s); a zero port is not an address", asked)
	}
}

// TestAddr_ReadsAsSomethingToActOn: the report has to name the node and what
// the port is for, because a bare number sends an operator to lsof.
func TestAddr_ReadsAsSomethingToActOn(t *testing.T) {
	a := inspector.Addr{Host: "10.0.0.3", Port: 31011, Node: 2, Purpose: "etcd"}
	if got, want := a.String(), "10.0.0.3:31011 (node2 etcd)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

// TestScan_LocalPortsAreTestedByBinding.
//
// A dial answers only if the listener accepts loopback connections, and a node
// that binds a wildcard socket may not: measured on a running network, lsof
// showed *:8600 held while a dial to 127.0.0.1:8600 was refused, so a
// dial-only scan called a busy port free and the launch failed anyway.
// Binding asks the kernel the same question the launch is about to ask.
func TestScan_LocalPortsAreTestedByBinding(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	port := ln.Addr().(*net.TCPAddr).Port

	busy := inspector.Ports(context.Background(), []inspector.Addr{
		{Host: "127.0.0.1", Port: port, Node: 1, Purpose: "http"},
	}, nil)
	if len(busy) != 1 {
		t.Fatalf("busy = %v; a port this process holds must read as busy", busy)
	}

	_ = ln.Close()
	if busy := inspector.Ports(context.Background(), []inspector.Addr{
		{Host: "127.0.0.1", Port: port, Node: 1, Purpose: "http"},
	}, nil); len(busy) != 0 {
		t.Fatalf("busy = %v; the port was released", busy)
	}
}
