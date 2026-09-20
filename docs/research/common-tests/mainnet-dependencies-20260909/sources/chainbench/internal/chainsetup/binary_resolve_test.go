package chainsetup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestBinaryFor_ResolvesPerNodeOverFallback locks the launch-time rule: a node
// naming a binary runs the path the workspace resolved for that name; a node
// naming none, or one whose name was never resolved, runs the single fallback.
func TestBinaryFor_ResolvesPerNodeOverFallback(t *testing.T) {
	w := &Workspace{state: State{Binaries: map[string]string{
		"wbft": "/opt/gwbft",
	}}}
	const fallback = "/opt/gstable"

	cases := map[string]struct {
		rec  node.Record
		want string
	}{
		"named and resolved":   {node.Record{Binary: "wbft"}, "/opt/gwbft"},
		"unnamed uses default": {node.Record{}, fallback},
		"named but unresolved": {node.Record{Binary: "ghost"}, fallback},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := w.binaryFor(tc.rec, fallback); got != tc.want {
				t.Errorf("binaryFor(%q) = %q, want %q", tc.rec.Binary, got, tc.want)
			}
		})
	}
}
