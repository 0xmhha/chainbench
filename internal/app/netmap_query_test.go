package app_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

// composeForQuery builds a small mixed network to ask questions about.
func composeForQuery(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	d := app.Deps{Clock: fixedClock}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
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
