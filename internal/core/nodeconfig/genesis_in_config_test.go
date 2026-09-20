package nodeconfig

import (
	"strings"
	"testing"
)

// TestTOML_ANodeCarriesItsGenesisAfterTheRest.
//
// A build that reads its genesis from the config file gets those tables last.
// They are [Eth.*] tables and everything the renderer writes above has already
// closed [Eth], so putting them anywhere but the end silently re-opens a table
// that the rest of the file is then written into.
func TestTOML_ANodeCarriesItsGenesisAfterTheRest(t *testing.T) {
	plain := string(TOML(Spec{}))
	if strings.Contains(plain, "Eth.Genesis") {
		t.Fatalf("an ordinary node's config talks about the genesis:\n%s", plain)
	}
	carried := string(TOML(Spec{Genesis: []byte("[Eth.Genesis]\nNonce = 66\n")}))
	if !strings.HasPrefix(carried, plain) {
		t.Errorf("carrying a genesis changed the rest of the config\n%s", carried)
	}
	if !strings.HasSuffix(strings.TrimRight(carried, "\n"), "Nonce = 66") {
		t.Errorf("the genesis is not last, so it re-opens [Eth] for whatever follows\n%s", carried)
	}
}
