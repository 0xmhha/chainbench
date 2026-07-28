// Package stablenet implements the chainbench capabilities specific to the
// stablenet chain (its governance system contracts), separate from the common
// set. Importing it for side effects loads stablenet.jsonl and registers its
// handlers into the capability registry.
package stablenet

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/pkg/mcp/capability"
)

//go:embed stablenet.jsonl
var catalog []byte

func init() {
	if err := capability.LoadCatalog(catalog); err != nil {
		panic(err)
	}
	reg := func(name string, h capability.Handler) {
		capability.RegisterHandler("v1", "stablenet", name, h)
	}
	reg("governance.propose_mint", proposeMint)
	reg("governance.propose_burn", proposeBurn)
	reg("governance.approve_proposal", byID(govbind.ApproveProposalCall))
	reg("governance.execute_proposal", byID(govbind.ExecuteProposalCall))
	reg("governance.cancel_proposal", byID(govbind.CancelProposalCall))
	reg("governance.disapprove_proposal", byID(govbind.DisapproveProposalCall))
	reg("governance.proposals", byID(govbind.ProposalsCall))
	reg("governance.propose_add_member", proposeAddMember)
	reg("governance.claim_burn_refund", claimBurnRefund)
}

func proposeMint(_ context.Context, args map[string]any) (string, error) {
	beneficiary := capability.ArgString(args, "beneficiary", "")
	amount, timestamp, err := amountTimestamp(args)
	if err != nil {
		return "", err
	}
	if beneficiary == "" {
		return "", fmt.Errorf("beneficiary is required")
	}
	proof := govbind.MintProof(beneficiary, amount, timestamp,
		capability.ArgString(args, "deposit_id", ""),
		capability.ArgString(args, "bank_reference", ""),
		capability.ArgString(args, "memo", ""))
	return govbind.ProposeMintCall(proof), nil
}

func proposeBurn(_ context.Context, args map[string]any) (string, error) {
	from := capability.ArgString(args, "from", "")
	amount, timestamp, err := amountTimestamp(args)
	if err != nil {
		return "", err
	}
	if from == "" {
		return "", fmt.Errorf("from is required")
	}
	proof := govbind.BurnProof(from, amount, timestamp,
		capability.ArgString(args, "withdrawal_id", ""),
		capability.ArgString(args, "reference_id", ""),
		capability.ArgString(args, "memo", ""))
	return govbind.ProposeBurnCall(proof), nil
}

func proposeAddMember(_ context.Context, args map[string]any) (string, error) {
	member := capability.ArgString(args, "member", "")
	if member == "" {
		return "", fmt.Errorf("member is required")
	}
	q := capability.ArgInt(args, "new_quorum", -1)
	if q < 0 {
		return "", fmt.Errorf("new_quorum is required (decimal)")
	}
	return govbind.ProposeAddMemberCall(member, uint32(q)), nil
}

func claimBurnRefund(_ context.Context, _ map[string]any) (string, error) {
	return govbind.ClaimBurnRefundCall(), nil
}

// byID adapts a govbind calldata builder that takes a single proposal id.
func byID(build func(*big.Int) string) capability.Handler {
	return func(_ context.Context, args map[string]any) (string, error) {
		id := capability.ArgBigInt(args, "id")
		if id == nil {
			return "", fmt.Errorf("id is required (decimal)")
		}
		return build(id), nil
	}
}

// amountTimestamp reads the shared amount/timestamp proof fields.
func amountTimestamp(args map[string]any) (amount, timestamp *big.Int, err error) {
	amount = capability.ArgBigInt(args, "amount")
	if amount == nil {
		return nil, nil, fmt.Errorf("amount is required (decimal wei)")
	}
	timestamp = capability.ArgBigInt(args, "timestamp")
	if timestamp == nil {
		return nil, nil, fmt.Errorf("timestamp is required (decimal)")
	}
	return amount, timestamp, nil
}
