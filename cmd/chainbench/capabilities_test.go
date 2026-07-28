package main

import (
	"strings"
	"testing"
)

func TestCapabilitiesCmd_All(t *testing.T) {
	out, err := run(t, "capabilities")
	if err != nil {
		t.Fatalf("capabilities: %v\n%s", err, out)
	}
	for _, want := range []string{
		"v1.common.chains.list",
		"v1.stablenet.governance.propose_mint",
		"v1.wemix.bootstrap.plan",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestCapabilitiesCmd_ScopedChain(t *testing.T) {
	out, err := run(t, "capabilities", "--chain", "stablenet")
	if err != nil {
		t.Fatalf("capabilities --chain stablenet: %v\n%s", err, out)
	}
	if !strings.Contains(out, "v1.common.chains.list") || !strings.Contains(out, "v1.stablenet.governance.propose_mint") {
		t.Errorf("stablenet scope should include common + stablenet:\n%s", out)
	}
	if strings.Contains(out, "v1.wemix.") {
		t.Errorf("stablenet scope must not include wemix capabilities:\n%s", out)
	}
}
