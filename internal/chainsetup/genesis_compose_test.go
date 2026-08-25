package chainsetup_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register the plugins
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

func typeName(v any) string { return strings.TrimPrefix(fmt.Sprintf("%T", v), "chainsetup.") }

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	return filepath.Join(append([]string{repoRoot(t)}, parts...)...)
}

func plugin(t *testing.T, id string) registry.ChainPlugin {
	t.Helper()
	p, err := registry.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestGenesisSourceFor_TheFamilyDecides: substituting a template for a family
// whose binary writes its own genesis produces a file that initializes cleanly
// and runs the wrong consensus, so the choice is not a caller's to make. It was
// being made in five places before this.
func TestGenesisSourceFor_TheFamilyDecides(t *testing.T) {
	for _, tc := range []struct{ chain, want string }{
		{"stablenet", "PresetGenesisSource"},
		{"wbft", "PresetGenesisSource"},
		{"wemix", "WemixGenesisSource"},
	} {
		got := chainsetup.GenesisSourceFor(plugin(t, tc.chain), chainsetup.GenesisConfig{})
		name := strings.TrimPrefix(strings.TrimPrefix(typeName(got), "chainsetup."), "*")
		if name != tc.want {
			t.Errorf("%s uses %s, want %s", tc.chain, name, tc.want)
		}
	}
}

// TestBuildGenesis_CustomizesWhateverTheFamilyProduced.
//
// The customization used to live inside the source that only one family uses,
// so a genesis overlay reached a wemix network through the step surface (which
// applied it separately, in its own copy of these steps) and was dropped on the
// engine one. Applying it outside the source means the caller's changes land on
// whatever base the family built.
func TestBuildGenesis_CustomizesWhateverTheFamilyProduced(t *testing.T) {
	art, err := chainsetup.BuildGenesis(context.Background(), plugin(t, "stablenet"),
		chainsetup.GenesisRequest{Validators: 4},
		chainsetup.GenesisConfig{
			KeysDir: repoPath(t, "keys", "preset"),
			Overlay: []byte(`{"config":{"aMarkerForThisTest":7}}`),
		})
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}
	var doc struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(art.Genesis, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Config["aMarkerForThisTest"]; !ok {
		t.Fatalf("the overlay did not reach the genesis: config = %v", doc.Config)
	}
}
