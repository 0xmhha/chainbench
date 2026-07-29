//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"
	"time"
)

// TestE2E_StablenetProposalExpiry is the proposal-expiry scenario (former
// stablenet-proposal-expiry.sh, regression f3-06): boot with the short-expiry
// genesis overlay (GovValidator proposal expiry shortened to ~30s + the
// "short-expiry" capability), then run the gated case that proposes, waits for
// expiry, and asserts the proposal transitions to Expired.
func TestE2E_StablenetProposalExpiry(t *testing.T) {
	bin := requireBinary(t, "GSTABLE_BIN", "gstable")
	cli := buildCLI(t)
	overlay := filepath.Join(repoRoot(t), "pkg", "chains", "stablenet", "overlays", "short-expiry.json")

	n := bootOverlay(t, cli, "stablenet", bin, 4, 1, overlay)

	// The overlay must have advertised its capability, else the gated case skips.
	if !contains(n.capabilities(), "short-expiry") {
		t.Fatalf("nodeset missing short-expiry capability — overlay not applied: %v", n.capabilities())
	}

	// Wait for the network to be up and producing before driving the case.
	n.waitAdvancing(n.rpcURL, 45*time.Second)

	// The case proposes, waits ~35s for expiry, and asserts Expired.
	n.runCase("proposal-expiry-transitions")
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
