package main

import "testing"

// Buckets follow layers.md §3: preset is an L1 primitive, testsupport is a test
// gate with no product code. A package that falls through to "other" shows up
// as a false violation whenever core imports it.
func TestLayerOfPlacesPackagesWhereLayersSays(t *testing.T) {
	for _, c := range []struct{ path, want string }{
		{"internal/preset", "core"},
		{"internal/resource", "core"},
		{"internal/core/node", "core"},
		{"internal/testsupport", "support"},
		{"internal/chainsetup/verb", "orchestration"},
		{"internal/app", "usecase"},
		{"cmd/chainbench", "entry"},
	} {
		if got := layerOf(c.path); got != c.want {
			t.Errorf("layerOf(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
