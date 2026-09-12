package stablenet

import (
	"bufio"
	"bytes"
	"encoding/json"
	"sort"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// catalogNames reads the embedded .jsonl the way LoadCatalog does, so the test
// sees the same declarations the registry was given.
func catalogNames(t *testing.T) []string {
	t.Helper()
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(catalog))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		b := bytes.TrimSpace(sc.Bytes())
		if len(b) == 0 || b[0] == '#' {
			continue
		}
		var d registry.Descriptor
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatalf("catalog line does not parse: %v", err)
		}
		out = append(out, d.Name)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// TestEveryDeclaredCapabilityIsExposed is the property this package had no test
// for: a capability the catalog declares and nobody bound a handler to is not an
// error anywhere. registry.All deliberately returns only EXPOSED capabilities —
// ones with a handler or a pre-existing flat tool — so an entry added to the
// .jsonl without its handler simply does not appear, and a caller asking for it
// is told it does not exist while the file says it does.
//
// The two lists agree today (ten and ten). Nothing held them together, which is
// the same shape as a comparison whose want side is never filled: both halves
// written, no seam.
func TestEveryDeclaredCapabilityIsExposed(t *testing.T) {
	declared := catalogNames(t)
	if len(declared) == 0 {
		t.Fatal("the embedded catalog declares nothing")
	}
	for _, name := range declared {
		addr := "v1.stablenet." + name
		cap, ok := registry.Lookup(addr)
		if !ok {
			t.Errorf("%s is declared in caps.jsonl and not exposed — bind a handler in init() or remove the entry", addr)
			continue
		}
		if cap.Handler == nil && cap.Tool == "" {
			t.Errorf("%s resolves to neither a handler nor a flat tool, so a caller cannot reach it", addr)
		}
	}
}

// The other direction cannot be tested from here, and saying so is better than a
// test that looks like it covers it.
//
// A handler bound to an address the catalog does not declare is dead: registry.All
// and registry.For both iterate the CATALOG and filter by exposure, so an orphan
// handler never appears in either. The bound-handler map is private and has no
// lister, which means the state is unobservable from outside the registry.
//
// A first version of this file asserted that direction anyway, by looping over
// registry.For("stablenet") and checking each name was declared. Removing a
// catalog line to test it left the suite green — the entry vanished from both
// sides at once, so the assertion could not fail. It was the shape it was written
// to prevent, and it is gone rather than left in looking like cover.
//
// The severity is also low, which is why no lister was added to core for it: an
// orphan handler is unreachable, not wrong. The direction that matters — a
// declaration nobody implemented, which a caller CAN ask for and be refused — is
// covered above.
