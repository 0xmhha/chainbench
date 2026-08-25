package netmap_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/netmap"
)

// TestEnode pins the URL byte for byte: a peer list is only compatible with
// what nodes already parse if the format never drifts.
func TestEnode(t *testing.T) {
	got := netmap.Enode("deadbeef", "127.0.0.1", 30303)
	want := "enode://deadbeef@127.0.0.1:30303?discport=0"
	if got != want {
		t.Errorf("Enode: got %q, want %q", got, want)
	}
}
