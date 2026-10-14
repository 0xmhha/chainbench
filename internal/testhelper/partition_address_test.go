package testhelper

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPartitionUsesRecordedReachableAddress(t *testing.T) {
	d := faultDeps(nil)
	rec := &peerRPC{}
	advertised := "enode://identity@203.0.113.9:30303?discport=0"
	srv := rec.server(t, advertised)
	env := envFromURLs(t, []string{srv.URL})
	nodes := env.Nodes()
	nodes[0].Ports.P2P = 31000
	got, err := enodesFor(context.Background(), &interp.ActionCtx{Deps: &d, Env: env}, nodes)
	want := "enode://identity@127.0.0.1:31000?discport=0"
	if err != nil || got[1] != want {
		t.Fatalf("peer uses NAT advertisement: got=%v err=%v want=%s", got, err, want)
	}
	// Attach endpoints lack a recorded P2P address; their explicit advertisement is retained.
	nodes[0].Ports.P2P = 0
	got, err = enodesFor(context.Background(), &interp.ActionCtx{Deps: &d, Env: env}, nodes)
	if err != nil || got[1] != advertised {
		t.Fatalf("attach advertisement changed: %v %v", got, err)
	}
}
