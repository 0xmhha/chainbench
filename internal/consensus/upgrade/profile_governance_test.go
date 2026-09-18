package upgrade

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
)

// TestProfile_TheGovernancePolicyIsTheDefault.
//
// The golden profile spells out thirteen governance parameters — ballot
// durations, staking bounds, block reward, fee ceiling, reward split. Every one
// of them is what poa.DefaultEnv already says, so the block restates the
// default rather than choosing anything.
//
// That is what keeps the governance policy off the list of things stopping a
// handoff from composing through the ordinary path: the ordinary genesis source
// takes an Env and defaults it to exactly this, so nothing has to be declared
// for the composition to produce the same chain.
//
// A failure here is not a mistake to undo. It means the profile has started
// choosing rather than restating, and that is precisely the point at which a
// governance policy needs a way to be DECLARED — a DSL surface for it, which is
// deliberately not built while nothing asks for one.
func TestProfile_TheGovernancePolicyIsTheDefault(t *testing.T) {
	prof, err := LoadProfile("../../../presets/hardfork/wemix-upgrade.yaml")
	if err != nil {
		t.Fatalf("read the golden profile: %v", err)
	}
	declared := prof.GovernanceEnv()

	if diff := envDiff(declared, poa.DefaultEnv()); diff != "" {
		t.Errorf("the profile's governance policy is no longer the default:\n%s\n"+
			"Nothing can declare a policy yet — this is where that surface becomes needed.", diff)
	}
}

// envDiff reports the fields two policies disagree on, as JSON so a big.Int
// reads as its number.
func envDiff(a, b poa.Env) string {
	ja, _ := json.MarshalIndent(a, "", "  ")
	jb, _ := json.MarshalIndent(b, "", "  ")
	if string(ja) == string(jb) {
		return ""
	}
	return "profile:\n" + string(ja) + "\ndefault:\n" + string(jb)
}
