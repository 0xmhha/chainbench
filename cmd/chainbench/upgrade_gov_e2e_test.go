//go:build e2e

// This E2E ports the wemix4 governance read cases (GOV-001 contract deploy,
// GOV-002 config params). The wemix governance system contracts are deployed at
// the Croissant fork block, so they live on the go-wbft SUCCESSOR chain (the
// go-wemix producer stops at croissant-1 and never sees them). It composes the
// handoff the way every test in this group does — runGovHandoff — and asserts
// governance on the successor. Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	go test -tags e2e -run TestWemixGovernanceE2E -timeout 8m ./cmd/chainbench
package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
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
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	// Original GOV-001 uses the actual 99/100 boundary. GOV-002 reads the
	// configured mainnet-scale values at that exact deployment block, not latest.
	c := rpc.Dial(runGovHandoffSpec(t, fromBin, toBin, writeGovernanceOriginalForkCase(t)))
	ctx := context.Background()
	p := governanceForkProof{Fork: 100, BeforeHeight: 99, AtHeight: 100, CodeBefore: map[string]string{}, CodeAt: map[string]string{}, Configured: governanceForkFixture().Configured}
	for _, addr := range []string{e2eGovConfig, e2eGovStaking, e2eGovRewardee, e2eGovNCP} {
		var before, at string
		if err := c.Call(ctx, "eth_getCode", &before, addr, "0x63"); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getCode", &at, addr, "0x64"); err != nil {
			t.Fatal(err)
		}
		p.CodeBefore[addr], p.CodeAt[addr] = before, at
	}
	for i, sig := range []string{"minimumStaking()", "maximumStaking()", "unbondingPeriodStaker()", "unbondingPeriodDelegator()"} {
		if err := c.Call(ctx, "eth_getStorageAt", &p.Storage[i], e2eGovConfig, fmt.Sprintf("0x%x", i), "0x64"); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_call", &p.Getters[i], map[string]string{"to": e2eGovConfig, "data": accounts.Selector(sig)}, "0x64"); err != nil {
			t.Fatal(err)
		}
	}
	logGovernanceProof(t, "GOV-001-002-original-fork", p)
	if err := verifyGovernanceFork(p); err != nil {
		t.Fatal(err)
	}

	// RPC-009: eth_call reaches a second governance contract — GovStaking's
	// totalStaking() decodes to a non-negative amount (>= 0; no stakers yet).
	out, err := c.EthCall(ctx, e2eGovStaking, accounts.Selector("totalStaking()"))
	if err != nil {
		t.Fatalf("eth_call GovStaking totalStaking(): %v", err)
	}
	if _, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16); !ok {
		t.Fatalf("RPC-009: GovStaking.totalStaking() not a uint (got %q)", out)
	}
}
