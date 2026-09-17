package upgrade

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProfile_PlanOrderPutsTheProducerOnTheAccountThatCanSeal.
//
// plan_order remaps which preset identity each plan node takes, and the reason
// is one slot: the preset's node5 carries a keystore for an account that is NOT
// the address its own nodekey derives. That account is the producer's — it is
// what the genesis funds and stakes and what the node unlocks to seal — so the
// producer has to land on node5 whatever else moves.
//
// This pins the two files to each other. The profile names the producer account
// and the preset holds the keystore for it; if either moves, the producer
// unlocks an account it has no key for and exits with "no key for given address
// or file", which says nothing about the profile that caused it.
//
// It is also the answer to whether plan_order is a blocker for composing a
// handoff through the ordinary path. It is not a remap the declaration cannot
// express: it exists because a handoff plan puts its producers FIRST while the
// preset's producer-capable entry is LAST. A node table chooses its own order,
// so it can put the producer on node5 and need no remap at all.
func TestProfile_PlanOrderPutsTheProducerOnTheAccountThatCanSeal(t *testing.T) {
	prof, err := LoadProfile("../../../presets/hardfork/wemix-upgrade.yaml")
	if err != nil {
		t.Fatalf("read the golden profile: %v", err)
	}
	order := prof.Identities.PlanOrder
	if len(order) == 0 {
		t.Fatal("plan_order is empty, so nothing says which identity the producer takes")
	}
	if len(prof.Producers.Members) == 0 {
		t.Fatal("the profile names no producer member")
	}

	// Plan node 1 is the producer; order[0] is the preset slot it takes.
	acct, err := keystoreAccount(t, "../../../keys/preset", order[0])
	if err != nil {
		t.Fatalf("read preset node%d's keystore: %v", order[0], err)
	}
	if want := prof.Producers.Members[0]; !strings.EqualFold(acct, want) {
		t.Errorf("preset node%d's keystore holds %s, but the profile stakes %s —\n"+
			"the producer would unlock an account it has no key for", order[0], acct, want)
	}
}

// keystoreAccount is the account one preset node's keystore file holds, which
// is not always the address that node's nodekey derives.
func keystoreAccount(t *testing.T, presetDir string, node int) (string, error) {
	t.Helper()
	dir := filepath.Join(presetDir, "node"+itoa(node), "keystore")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			return "", rerr
		}
		var doc struct {
			Address string `json:"address"`
		}
		if json.Unmarshal(b, &doc) == nil && doc.Address != "" {
			return "0x" + doc.Address, nil
		}
	}
	return "", os.ErrNotExist
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
