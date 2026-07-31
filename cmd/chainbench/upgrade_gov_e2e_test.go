//go:build e2e

// This E2E ports the wemix4 governance read cases (GOV-001 contract deploy,
// GOV-002 config params). The wemix governance system contracts are deployed at
// the Croissant fork block, so they live on the go-wbft SUCCESSOR chain (the
// go-wemix producer stops at croissant-1 and never sees them). It drives the same
// `chainbench upgrade run` handoff as TestUpgradeRunE2E and asserts governance on
// the successor. Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestWemixGovernanceE2E -timeout 8m ./cmd/chainbench
package main

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

// Governance system-contract addresses (fixed by the wemix bootstrap).
const (
	e2eGovConfig   = "0x0000000000000000000000000000000000001000"
	e2eGovStaking  = "0x0000000000000000000000000000000000001001"
	e2eGovRewardee = "0x0000000000000000000000000000000000001002"
	e2eGovNCP      = "0x0000000000000000000000000000000000001003"
)

func TestWemixGovernanceE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	// Governance lives on the go-wbft successor (deployed at the fork block).
	c := rpc.Dial(runGovHandoff(t, fromBin, toBin, template))
	ctx := context.Background()

	// GOV-001: all four governance system contracts carry code.
	for _, addr := range []string{e2eGovConfig, e2eGovStaking, e2eGovRewardee, e2eGovNCP} {
		code, err := c.CodeAt(ctx, addr)
		if err != nil {
			t.Fatalf("eth_getCode %s: %v", addr, err)
		}
		if code == "" || code == "0x" {
			t.Fatalf("GOV-001: no governance contract deployed at %s", addr)
		}
	}

	// GOV-002: GovConfig returns positive values for its core parameters.
	for _, sig := range []string{
		"minimumStaking()",
		"unbondingPeriodStaker()",
		"unbondingPeriodDelegator()",
		"changeFeeDelay()",
	} {
		out, err := c.EthCall(ctx, e2eGovConfig, accounts.Selector(sig))
		if err != nil {
			t.Fatalf("eth_call GovConfig %s: %v", sig, err)
		}
		v, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
		if !ok || v.Sign() <= 0 {
			t.Fatalf("GOV-002: GovConfig.%s must be > 0 (got %q)", sig, out)
		}
	}
}
