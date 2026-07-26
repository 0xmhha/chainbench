// This file ports the go-stablenet governance mint-proposal lifecycle (from
// tests/regression f-system-contracts f2-01), exercising node-side signing (a
// validator's unlocked coinbase votes) plus the govbind calldata/decoders.
//
// # Test: mint-proposal-executes
//
// Intent:   a GovMinter mint proposal, proposed and approved by validators to
//
//	quorum, executes and credits the beneficiary with the minted amount.
//
// Applies:  stablenet (the go-stablenet GovBase system contracts). Requires "rpc".
// Method:   discover each validator's coinbase (eth_coinbase); propose
//
//	proposeMint(proof) from the first validator; extract the proposalId from
//	the ProposalCreated log; approve from the other validators until the
//	proposals() status reads Executed (a quorum approval auto-executes),
//	executing manually if needed.
//
// Pass:     the beneficiary's balance increases by exactly the minted amount.
//
// This is chainbench TEST CODE (requirement #16): it drives real governance
// transactions signed node-side by validators, so it is only meaningful against
// a live multi-validator network (the sibling _test.go validates
// registration/gating). It needs at least two validators for quorum.
package anzeon

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// govMinter is the go-stablenet GovMinter system contract (regression
// lib/common.sh GOV_MINTER).
const govMinter = "0x0000000000000000000000000000000000001003"

// mintBeneficiary starts unfunded so the minted amount is unambiguous.
const mintBeneficiary = "0x00000000000000000000000000000000C0FFEE05"

// govGas is a generous gas cap for the governance calls (regression uses 1.5M).
const govGas = "0x16e360"

func init() {
	testkit.Register(testkit.Case{
		Name:         "mint-proposal-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           mintProposalExecutes,
	})
}

// validator pairs a node's RPC client with its unlocked coinbase (the account
// that signs its governance votes node-side).
type validator struct {
	client *rpc.Client
	addr   string
}

func mintProposalExecutes(t *testkit.T) {
	ctx := t.Ctx()
	var vals []validator
	for _, n := range t.NodeSet().Nodes {
		if n.Role != node.RoleValidator {
			continue
		}
		c := rpc.Dial(n.RPCURL)
		cb, err := c.Coinbase(ctx)
		t.NoErr(err, "eth_coinbase")
		vals = append(vals, validator{client: c, addr: cb})
	}
	t.Truef(len(vals) >= 2, "need >=2 validators for quorum, got %d", len(vals))
	proposer := vals[0]

	balBefore, err := proposer.client.BalanceAt(ctx, mintBeneficiary)
	t.NoErr(err, "balance before")

	amount := big.NewInt(1_000_000_000_000_000_000) // 1 coin
	proof := govbind.MintProof(mintBeneficiary, amount, big.NewInt(time.Now().Unix()),
		"REG-DEP", "REG-BANK", "regression mint")
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeMintCall(proof), Gas: govGas,
	})
	t.NoErr(err, "proposeMint")

	// Extract the proposalId once the propose tx is mined.
	var proposalID *big.Int
	t.WaitFor(func() bool {
		logs, ok := receiptLogs(ctx, proposer.client, proposeHash)
		if !ok {
			return false
		}
		proposalID, ok = govbind.ExtractProposalID(logs)
		return ok
	}, 60*time.Second, time.Second, "proposalId from ProposalCreated log")

	// Approve from each other validator until the proposal executes (a quorum
	// approval auto-executes inside the approve tx).
	executed := false
	for _, v := range vals[1:] {
		apHash, err := v.client.SendTransaction(ctx, rpc.SendTxArgs{
			From: v.addr, To: govMinter, Data: govbind.ApproveProposalCall(proposalID), Gas: govGas,
		})
		t.NoErr(err, "approveProposal")
		t.WaitFor(func() bool { return receiptSucceeded(ctx, v.client, apHash) },
			60*time.Second, time.Second, "approve receipt")
		if proposalExecuted(ctx, proposer.client, proposalID) {
			executed = true
			break
		}
	}
	if !executed {
		exHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
			From: proposer.addr, To: govMinter, Data: govbind.ExecuteProposalCall(proposalID), Gas: govGas,
		})
		t.NoErr(err, "executeProposal")
		t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, exHash) },
			60*time.Second, time.Second, "execute receipt")
	}

	// The beneficiary is credited the minted amount.
	t.WaitFor(func() bool {
		balAfter, err := proposer.client.BalanceAt(ctx, mintBeneficiary)
		if err != nil {
			return false
		}
		return new(big.Int).Sub(balAfter, balBefore).Cmp(amount) == 0
	}, 60*time.Second, time.Second, "beneficiary credited the minted amount")
}

// receiptLogs returns a successful tx's logs, and false while it is pending or
// reverted.
func receiptLogs(ctx context.Context, c *rpc.Client, hash string) ([]accounts.Log, bool) {
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	var r struct {
		Status string         `json:"status"`
		Logs   []accounts.Log `json:"logs"`
	}
	if err := json.Unmarshal(raw, &r); err != nil || r.Status != "0x1" {
		return nil, false
	}
	return r.Logs, true
}

// receiptSucceeded reports whether a tx is mined with status 0x1.
func receiptSucceeded(ctx context.Context, c *rpc.Client, hash string) bool {
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil || len(raw) == 0 {
		return false
	}
	var r struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(raw, &r) == nil && r.Status == "0x1"
}

// proposalExecuted reads the proposals() status and reports whether it is Executed.
func proposalExecuted(ctx context.Context, c *rpc.Client, id *big.Int) bool {
	ret, err := c.EthCall(ctx, govMinter, govbind.ProposalsCall(id))
	if err != nil {
		return false
	}
	st, ok := govbind.DecodeProposalStatus(ret)
	return ok && st == govbind.ProposalExecuted
}
