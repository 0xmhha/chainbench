// Shared helpers for the go-stablenet governance write cases: validator
// discovery via node-side signing (each validator votes from its own unlocked
// coinbase) and receipt/proposal-status reads. Used by the mint / burn /
// validator proposal cases in this package.
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

// go-stablenet GovBase system contracts (regression lib/common.sh).
const (
	govMinter    = "0x0000000000000000000000000000000000001003"
	govValidator = "0x0000000000000000000000000000000000001001"
)

// govGas is a generous gas cap for the governance calls (regression uses 1.5M).
const govGas = "0x16e360"

// validator pairs a node's RPC client with its unlocked coinbase (the account
// that signs its governance votes node-side).
type validator struct {
	client *rpc.Client
	addr   string
}

// discoverValidators returns a client+coinbase for each validator node, failing
// unless at least two exist (governance needs a quorum).
func discoverValidators(t *testkit.T) []validator {
	var vals []validator
	for _, n := range t.NodeSet().Nodes {
		if n.Role != node.RoleValidator {
			continue
		}
		c := rpc.Dial(n.RPCURL)
		cb, err := c.Coinbase(t.Ctx())
		t.NoErr(err, "eth_coinbase")
		vals = append(vals, validator{client: c, addr: cb})
	}
	t.Truef(len(vals) >= 2, "need >=2 validators for quorum, got %d", len(vals))
	return vals
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

// proposalExecuted reads contract.proposals(id) and reports whether the status
// is Executed.
func proposalExecuted(ctx context.Context, c *rpc.Client, contract string, id *big.Int) bool {
	ret, err := c.EthCall(ctx, contract, govbind.ProposalsCall(id))
	if err != nil {
		return false
	}
	st, ok := govbind.DecodeProposalStatus(ret)
	return ok && st == govbind.ProposalExecuted
}

// extractProposalID waits for a propose tx to mine and returns the proposalId
// from its ProposalCreated log.
func extractProposalID(t *testkit.T, v validator, proposeHash string) *big.Int {
	var id *big.Int
	t.WaitFor(func() bool {
		logs, ok := receiptLogs(t.Ctx(), v.client, proposeHash)
		if !ok {
			return false
		}
		id, ok = govbind.ExtractProposalID(logs)
		return ok
	}, 60*time.Second, time.Second, "proposalId from ProposalCreated log")
	return id
}

// approveToQuorum approves proposalID on contract from every validator after the
// proposer until the proposal executes (a quorum approval auto-executes), then
// executes manually if it did not. proposer executes; approvers vote.
func approveToQuorum(t *testkit.T, contract string, proposalID *big.Int, proposer validator, approvers []validator) {
	ctx := t.Ctx()
	for _, v := range approvers {
		apHash, err := v.client.SendTransaction(ctx, rpc.SendTxArgs{
			From: v.addr, To: contract, Data: govbind.ApproveProposalCall(proposalID), Gas: govGas,
		})
		t.NoErr(err, "approveProposal")
		t.WaitFor(func() bool { return receiptSucceeded(ctx, v.client, apHash) },
			60*time.Second, time.Second, "approve receipt")
		if proposalExecuted(ctx, proposer.client, contract, proposalID) {
			return
		}
	}
	exHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: contract, Data: govbind.ExecuteProposalCall(proposalID), Gas: govGas,
	})
	t.NoErr(err, "executeProposal")
	t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, exHash) },
		60*time.Second, time.Second, "execute receipt")
}
