package inspector_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/inspector"
)

// TestHosts_ReportsTheUnreachable: a host whose door does not answer is
// down; this machine is never dialled.
func TestHosts_ReportsTheUnreachable(t *testing.T) {
	dial := func(_ context.Context, hostPort string) (net.Conn, error) {
		if hostPort == "10.0.0.2:22" {
			return nil, errors.New("no route")
		}
		c1, c2 := net.Pipe()
		go func() { _ = c2.Close() }()
		return c1, nil
	}
	down := inspector.Hosts(context.Background(), []inspector.Host{
		{Name: "local", Addr: "127.0.0.1", Port: 22},
		{Name: "a", Addr: "10.0.0.1", Port: 22},
		{Name: "b", Addr: "10.0.0.2", Port: 22},
	}, dial)
	if len(down) != 1 || down[0].Name != "b" {
		t.Fatalf("down = %+v, want only b", down)
	}
}
