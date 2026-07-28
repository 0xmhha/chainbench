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

func TestCapabilitiesCall_ChainSpecific(t *testing.T) {
	// A handler-backed chain-specific capability is invokable and returns
	// calldata (pure, no node needed).
	out, err := run(t, "capabilities", "call", "v1.stablenet.governance.propose_mint",
		"--arg", "beneficiary=0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
		"--arg", "amount=1000000000000000000",
		"--arg", "timestamp=1700000000",
	)
	if err != nil {
		t.Fatalf("call: %v\n%s", err, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "0x") {
		t.Errorf("expected calldata, got %q", out)
	}
}

func TestCapabilitiesCall_Wemix(t *testing.T) {
	out, err := run(t, "capabilities", "call", "v1.wemix.bootstrap.plan")
	if err != nil {
		t.Fatalf("call: %v\n%s", err, out)
	}
	if !strings.Contains(out, "deploy-governance") {
		t.Errorf("expected bootstrap plan, got %q", out)
	}
}

func TestCapabilitiesCall_Common(t *testing.T) {
	// A handler-backed common capability is invokable from the CLI.
	out, err := run(t, "capabilities", "call", "v1.common.chains.list")
	if err != nil {
		t.Fatalf("call: %v\n%s", err, out)
	}
	if !strings.Contains(out, "stablenet") {
		t.Errorf("expected chains listing, got %q", out)
	}
}

func TestCapabilitiesCall_Errors(t *testing.T) {
	if _, err := run(t, "capabilities", "call", "v1.common.nope"); err == nil {
		t.Error("expected unknown-capability error")
	}
	if _, err := run(t, "capabilities", "call", "v1.stablenet.governance.propose_mint", "--arg", "bad"); err == nil {
		t.Error("expected bad --arg error")
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
