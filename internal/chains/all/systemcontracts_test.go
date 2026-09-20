package all_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// contractBlocks are the keys a genesis template puts its own contracts under.
// stablenet calls the block systemContracts and wbft calls it govContracts;
// nothing else about them differs.
var contractBlocks = []string{"systemContracts", "govContracts"}

// quotedPlaceholder and barePlaceholder match the template's substitution
// tokens. They come in two shapes — inside a string ("__SC_MEMBERS_CSV__") and
// in value position (__CHAIN_ID__, __ALLOC_JSON__) — so a template is not valid
// JSON until both are stood in for.
var (
	quotedPlaceholder = regexp.MustCompile(`"__[A-Z0-9_]+__"`)
	barePlaceholder   = regexp.MustCompile(`__[A-Z0-9_]+__`)
)

// address matches a 20-byte address written out.
var address = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// TestManifest_SystemContractsMatchTheGenesisTemplate.
//
// The manifest is where a case will ask "does this chain have a govMinter", and
// the template is what actually puts one in the genesis. Two copies of a fact
// drift, and this one drifts silently: a manifest that names the wrong address
// sends a test to whatever else happens to be there, and on these chains
// something else usually is — 0x…1001 is govValidator on stablenet and
// govStaking on wbft.
//
// A manifest entry with no counterpart in the template is allowed and checked
// only for shape. A chain may deploy its contracts at run time, and a contract
// the binary provides is in no template: stablenet's accountManager is the
// standing example.
func TestManifest_SystemContractsMatchTheGenesisTemplate(t *testing.T) {
	for _, id := range registry.Names() {
		t.Run(id, func(t *testing.T) {
			p, err := registry.Get(id)
			if err != nil {
				t.Fatal(err)
			}
			declared := p.Manifest().SystemContracts
			for name, addr := range declared {
				if !address.MatchString(addr) {
					t.Errorf("%s: %s is %q, which is not an address", id, name, addr)
				}
			}
			inTemplate := templateContracts(t, p.GenesisTemplate())
			if len(inTemplate) == 0 {
				return // no template: the manifest is the only statement there is
			}
			for name, addr := range inTemplate {
				got, ok := declared[name]
				if !ok {
					t.Errorf("%s: the genesis template puts %s at %s and the manifest does not name it", id, name, addr)
					continue
				}
				if !strings.EqualFold(got, addr) {
					t.Errorf("%s: %s is %s in the manifest and %s in the genesis template", id, name, got, addr)
				}
			}
		})
	}
}

// templateContracts reads the name-to-address pairs out of a genesis template's
// contract block, or nil when the chain has no template.
func templateContracts(t *testing.T, tmpl []byte) map[string]string {
	t.Helper()
	if len(tmpl) == 0 {
		return nil
	}
	// Stand in for the substitution tokens so the document parses. The values
	// are never read — only the contract block is — so anything valid will do.
	filled := quotedPlaceholder.ReplaceAll(tmpl, []byte(`""`))
	filled = barePlaceholder.ReplaceAll(filled, []byte(`0`))

	var doc any
	if err := json.Unmarshal(filled, &doc); err != nil {
		t.Fatalf("genesis template does not parse once its placeholders are filled: %v", err)
	}
	out := map[string]string{}
	collectContracts(doc, out)
	return out
}

// collectContracts finds every contract block in the document and records the
// address each named child sits at.
func collectContracts(node any, out map[string]string) {
	m, ok := node.(map[string]any)
	if !ok {
		if list, isList := node.([]any); isList {
			for _, v := range list {
				collectContracts(v, out)
			}
		}
		return
	}
	for _, block := range contractBlocks {
		entries, ok := m[block].(map[string]any)
		if !ok {
			continue
		}
		for name, entry := range entries {
			e, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if addr, ok := e["address"].(string); ok && address.MatchString(addr) {
				out[name] = addr
			}
		}
	}
	for _, v := range m {
		collectContracts(v, out)
	}
}
