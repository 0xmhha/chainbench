package upgrade_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/consensus/upgrade"
)

type fakePeerCaller struct {
	calls []string // "endpoint|enode"
	fail  map[string]bool
}

func (f *fakePeerCaller) AddPeer(_ context.Context, endpoint, enode string) error {
	if f.fail[endpoint] {
		return fmt.Errorf("boom")
	}
	f.calls = append(f.calls, endpoint+"|"+enode)
	return nil
}

func TestWireMesh_FullMesh(t *testing.T) {
	eps := []string{"http://a", "http://b", "http://c"}
	enodes := []string{"enA", "enB", "enC"}
	f := &fakePeerCaller{}
	if err := upgrade.WireMesh(context.Background(), f, eps, enodes); err != nil {
		t.Fatal(err)
	}
	// each of 3 nodes dials the other 2 -> 6 calls, none to itself.
	if len(f.calls) != 6 {
		t.Fatalf("want 6 addPeer calls, got %d: %v", len(f.calls), f.calls)
	}
	sort.Strings(f.calls)
	for _, c := range f.calls {
		ep, en, _ := strings.Cut(c, "|")
		self := map[string]string{"http://a": "enA", "http://b": "enB", "http://c": "enC"}
		if self[ep] == en {
			t.Errorf("node dialed itself: %s", c)
		}
	}
}

func TestWireMesh_SkipsEmptyAndReportsFailures(t *testing.T) {
	eps := []string{"http://a", "", "http://c"}
	enodes := []string{"enA", "enB", ""} // node2 endpoint empty, node3 enode empty
	f := &fakePeerCaller{fail: map[string]bool{"http://c": true}}
	err := upgrade.WireMesh(context.Background(), f, eps, enodes)
	// http://c fails on every addPeer it attempts -> error reported, not panics.
	if err == nil || !strings.Contains(err.Error(), "failure") {
		t.Fatalf("expected failure report, got %v", err)
	}
	// node with empty endpoint made no calls; nobody dials node3 (empty enode).
	for _, c := range f.calls {
		if strings.HasPrefix(c, "|") || strings.HasSuffix(c, "|") {
			t.Errorf("empty endpoint/enode used: %s", c)
		}
	}
	// length mismatch is rejected.
	if err := upgrade.WireMesh(context.Background(), f, []string{"a"}, []string{"x", "y"}); err == nil {
		t.Error("length mismatch should error")
	}
}
