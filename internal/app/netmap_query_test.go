package app_test

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/node"
)

const presetDir = "../../keys/preset"

// composeForQuery builds a small mixed network to ask questions about.
func composeForQuery(t *testing.T) string {
	t.Helper()
	// wemix, deliberately: it is the family whose derived etcd ports exist,
	// and the derived-port questions below are exactly the ones a wemix bind
	// failure raises. A wbft network carries no etcd port to ask about.
	return composeChainForQuery(t, "wemix")
}

func composeChainForQuery(t *testing.T, chain string) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: chain, KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{DataDir: dir, Validators: 3, Endpoints: 1}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	return dir
}

// TestNetMap_AnswersInBothDirections is the point of keeping a map rather than
// a list: an address in a log line has to lead back to a node, and a role in a
// test definition has to lead to an address.
func TestNetMap_AnswersInBothDirections(t *testing.T) {
	dir := composeForQuery(t)
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}

	all, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if all.Total != 4 || all.Roles["bp"] != 3 || all.Roles["en"] != 1 {
		t.Fatalf("map = %d nodes %v", all.Total, all.Roles)
	}
	// Every node carries both names.
	for _, e := range all.Entries {
		if e.Label == "" || e.Alias == "" {
			t.Fatalf("node%d is missing a name: %+v", e.Node, e)
		}
	}
	// The etcd port is derived, not launched — and it must still be answerable.
	first := all.Entries[0]
	if first.Etcd != first.P2P+1 {
		t.Fatalf("node1 etcd = %d, want p2p+1 (%d)", first.Etcd, first.P2P+1)
	}

	// Forward: the role alias a spec would use.
	byAlias, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Label: "en1"})
	if err != nil {
		t.Fatalf("map by alias: %v", err)
	}
	if len(byAlias.Entries) != 1 || byAlias.Entries[0].Role != "en" {
		t.Fatalf("en1 = %+v", byAlias.Entries)
	}
	// Forward: the identity an artifact would carry.
	byLabel, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Label: "node2"})
	if err != nil || len(byLabel.Entries) != 1 || byLabel.Entries[0].Node != 2 {
		t.Fatalf("node2 = %+v, %v", byLabel.Entries, err)
	}

	// Reverse: a port back to its owner, including the derived etcd port that
	// used to be unknowable once a network was running.
	byPort, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Port: first.Etcd})
	if err != nil {
		t.Fatalf("map by port: %v", err)
	}
	if len(byPort.Entries) != 1 || byPort.Entries[0].Node != first.Node {
		t.Fatalf("port %d = %+v, want node%d", first.Etcd, byPort.Entries, first.Node)
	}
	// Reverse: an address to everything on it.
	byHost, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Host: first.Host})
	if err != nil || len(byHost.Entries) != 4 {
		t.Fatalf("host %s = %d entries, %v", first.Host, len(byHost.Entries), err)
	}
}

func TestNetMap_RefusesAmbiguousAndUnknownQuestions(t *testing.T) {
	dir := composeForQuery(t)
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}

	// Two selectors ask two questions; honouring one silently would answer the
	// one the caller did not mean.
	if _, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Node: 1, Port: 8600}); err == nil {
		t.Fatal("two selectors must be refused")
	}
	if _, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Label: "sideways1"}); err == nil {
		t.Fatal("a label that is neither an identity nor a role must be refused")
	}
	// A question with no answer says so, rather than returning an empty map
	// that reads as "nothing is listening there".
	_, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Port: 65000})
	if err == nil || !strings.Contains(err.Error(), "nothing matches") {
		t.Fatalf("unmatched port error = %v", err)
	}
}

// TestNetPool_ReportsCapacityWithoutCredentials: the pool answers "why was that
// refused" — and must not become the place a password reaches an agent
// transcript. The absence is asserted, since a leak here would be silent.
func TestNetPool_ReportsCapacityWithoutCredentials(t *testing.T) {
	dir := composeForQuery(t)
	out, err := app.NetPool(context.Background(), app.Deps{Clock: fixedClock}, app.NetPoolIn{DataDir: dir})
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if out.Cap <= 0 || out.Slots <= 0 || len(out.Hosts) == 0 {
		t.Fatalf("pool = %+v", out)
	}
	if out.Used != 4 {
		t.Fatalf("used = %d, want the 4 composed nodes", out.Used)
	}
	if out.Source == "" {
		t.Fatal("the pool must say where the port plan came from")
	}
	// The type carries no credential field at all: adding one would have to be
	// a deliberate act that fails this test first.
	for _, field := range []string{"Password", "KeyFile", "User", "SSH"} {
		if hasField(out, field) {
			t.Fatalf("NetPoolOut exposes %s — credentials do not belong in a summary", field)
		}
	}
}

func hasField(v any, name string) bool {
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Name == name {
			return true
		}
	}
	return false
}

// TestNetMap_TracesAnAddressFromALogLine is the question a map exists to
// answer: a bind failure or a peer error prints host:port, and that has to lead
// back to a node without the operator matching numbers by eye. It answers for
// the derived etcd port too, which is exactly the one a wemix bind failure
// names and the one nothing could resolve before.
func TestNetMap_TracesAnAddressFromALogLine(t *testing.T) {
	dir := composeForQuery(t)
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}

	all, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	want := all.Entries[1]
	addr := fmt.Sprintf("%s:%d", want.Host, want.Etcd)

	got, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Addr: addr})
	if err != nil {
		t.Fatalf("map by addr %s: %v", addr, err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Node != want.Node {
		t.Fatalf("addr %s = %+v, want node%d", addr, got.Entries, want.Node)
	}
	// A malformed address says so rather than matching nothing quietly.
	if _, err := app.NetMap(ctx, d, app.NetMapIn{DataDir: dir, Addr: "127.0.0.1"}); err == nil {
		t.Fatal("an address without a port must be refused")
	}
}

// TestNetAllocate_PersistsTheLabel: the label is stored, and every path is
// named after it. Deriving it at each read is how four different spellings of a
// node's name came to exist.
func TestNetAllocate_PersistsTheLabel(t *testing.T) {
	dir := composeForQuery(t)
	out, err := app.NetStatus(context.Background(), app.Deps{Clock: fixedClock}, app.NetStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, n := range out.State.Nodes {
		if n.Label == "" {
			t.Fatalf("node%d has no persisted label", n.Index)
		}
		if !strings.HasSuffix(n.DataDir, n.Label) {
			t.Fatalf("node%d datadir %q is not named after %q", n.Index, n.DataDir, n.Label)
		}
		if !strings.HasSuffix(n.LogPath, n.Label+".log") {
			t.Fatalf("node%d log %q is not named after %q", n.Index, n.LogPath, n.Label)
		}
	}
	// A workspace written before the field still resolves: the label falls back
	// to the conventional one for its index.
	old := out.State.Nodes[0]
	old.Label = ""
	if got := old.NodeLabel(); got != "node1" {
		t.Fatalf("fallback label = %q, want node1", got)
	}
}

// TestMapEntry_EmbedsThePortSet guards a mistake this series made five times:
// a hand-written copy of a node's ports that omits one. Every previous instance
// was found by an operator noticing a missing number, once in this very type —
// it dropped the etcd client port the moment a family started reserving one.
//
// Embedding is what makes the omission impossible, so the embedding is what is
// asserted. Listing the fields again would restore the failure mode.
func TestMapEntry_EmbedsThePortSet(t *testing.T) {
	et := reflect.TypeOf(app.MapEntry{})
	for i := 0; i < et.NumField(); i++ {
		f := et.Field(i)
		if f.Anonymous && f.Type == reflect.TypeOf(node.Endpoints{}) {
			return
		}
	}
	t.Fatal("MapEntry must embed node.Endpoints — a copied port list will lose a port")
}
