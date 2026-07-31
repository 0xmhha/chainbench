//go:build e2e

// This E2E ports the first wemix4 governance WRITE flow (GOV-006: NCP add
// proposal + vote). On the go-wbft successor, the post-fork validator set's
// governance has one NCP (Node Change Proposal member) — the preset node-1
// account, whose raw key ships in keys/preset. As that sole NCP (quorum
// ceil(2*1/3) = 1), it proposes adding a new NCP and votes it through, and the
// new address becomes an NCP (ncpCount 1 -> 2). This exercises the GovNCP
// propose+vote+execute path end to end. Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestWemixGovernanceNCPAddE2E -timeout 8m ./cmd/chainbench
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

func TestWemixGovernanceNCPAddE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	url := runGovHandoff(t, fromBin, toBin, template)
	c := rpc.Dial(url)
	ctx := context.Background()

	// The sole NCP is the preset node-1 account; open its wallet on the successor.
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	w, err := ap.OpenWallet(ctx, presetNode1Key(t), url)
	if err != nil {
		t.Fatalf("open NCP wallet: %v", err)
	}

	const newNCP = "0x00000000000000000000000000000000CAFE0006"

	// Pre-state: the candidate is not yet an NCP.
	if ncpIsMember(t, c, newNCP) {
		t.Fatalf("candidate %s is already an NCP before the proposal", newNCP)
	}
	before := ncpCount(t, c)

	// Propose adding the new NCP; the receipt's first log topic[1] is the ballot id.
	rcpt := ncpExecute(t, c, w, accounts.EncodeCallArgs("newProposalToAddNCP(address)", accounts.Address(newNCP)))
	if rcpt.Status != "0x1" {
		t.Fatalf("newProposalToAddNCP reverted (status %s)", rcpt.Status)
	}
	if len(rcpt.Logs) == 0 || len(rcpt.Logs[0].Topics) < 2 {
		t.Fatalf("no ballot id in proposal receipt logs: %+v", rcpt.Logs)
	}
	ballot, ok := new(big.Int).SetString(strings.TrimPrefix(rcpt.Logs[0].Topics[1], "0x"), 16)
	if !ok {
		t.Fatalf("ballot id not hex: %s", rcpt.Logs[0].Topics[1])
	}

	// Vote it through (as the sole NCP, one accept vote meets quorum).
	vote := ncpExecute(t, c, w, accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(ballot), accounts.Word([]byte{1})))
	if vote.Status != "0x1" {
		t.Fatalf("vote reverted (status %s)", vote.Status)
	}

	// Post-state: the candidate is now an NCP and the count grew by one.
	if !ncpIsMember(t, c, newNCP) {
		t.Fatalf("candidate %s is not an NCP after the accepted proposal", newNCP)
	}
	if after := ncpCount(t, c); after.Cmp(new(big.Int).Add(before, big.NewInt(1))) != 0 {
		t.Fatalf("ncpCount did not grow by one: before=%s after=%s", before, after)
	}
}

// ncpReceipt is the subset of a tx receipt the NCP flow reads.
type ncpReceipt struct {
	Status string `json:"status"`
	Logs   []struct {
		Topics []string `json:"topics"`
	} `json:"logs"`
}

// ncpExecute sends calldata to GovNCP from w and returns the mined receipt.
func ncpExecute(t *testing.T, c *rpc.Client, w accounts.Wallet, data string) ncpReceipt {
	t.Helper()
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, err := w.Execute(context.Background(), e2eGovNCP, raw, nil)
	if err != nil {
		t.Fatalf("GovNCP execute: %v", err)
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		rc, err := c.TxReceipt(context.Background(), hash)
		if err == nil && len(rc) > 0 && string(rc) != "null" {
			var r ncpReceipt
			if json.Unmarshal(rc, &r) == nil && r.Status != "" {
				return r
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("GovNCP tx %s never mined", hash)
	return ncpReceipt{}
}

// ncpIsMember reports GovNCP.isNCP(addr).
func ncpIsMember(t *testing.T, c *rpc.Client, addr string) bool {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovNCP, accounts.EncodeCallArgs("isNCP(address)", accounts.Address(addr)))
	if err != nil {
		t.Fatalf("eth_call isNCP: %v", err)
	}
	v, _ := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	return v != nil && v.Sign() > 0
}

// ncpCount reads GovNCP.ncpCount().
func ncpCount(t *testing.T, c *rpc.Client) *big.Int {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovNCP, accounts.EncodeCallArgs("ncpCount()"))
	if err != nil {
		t.Fatalf("eth_call ncpCount: %v", err)
	}
	v, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	if !ok {
		t.Fatalf("ncpCount not hex: %s", out)
	}
	return v
}
