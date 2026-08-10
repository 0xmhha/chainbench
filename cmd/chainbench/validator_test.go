package main

import (
	"strings"
	"testing"
)

func TestValidatorRoster_WbftFamily(t *testing.T) {
	out, err := run(t, "validator", "roster", "--chain", "stablenet", "--keys", "../../keys/preset")
	if err != nil {
		t.Fatalf("validator roster: %v\n%s", err, out)
	}
	for _, want := range []string{"family: wbft", "validator", "BLS present", "governance-member"} {
		if !strings.Contains(out, want) {
			t.Fatalf("roster missing %q:\n%s", want, out)
		}
	}
}

func TestValidatorRoster_PoaFamilyNote(t *testing.T) {
	out, err := run(t, "validator", "roster", "--chain", "wemix", "--keys", "../../keys/preset")
	if err != nil {
		t.Fatalf("validator roster wemix: %v\n%s", err, out)
	}
	if !strings.Contains(out, "family: poa") || !strings.Contains(out, "bootstrap") {
		t.Fatalf("wemix roster should be poa + bootstrap note:\n%s", out)
	}
	if strings.Contains(out, "\nvalidator ") {
		t.Fatalf("poa roster must not list genesis validators:\n%s", out)
	}
}

func TestValidatorRoster_RequiresChain(t *testing.T) {
	if _, err := run(t, "validator", "roster", "--keys", "../../keys/preset"); err == nil {
		t.Fatal("expected error without --chain")
	}
}
