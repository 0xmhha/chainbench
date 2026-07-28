// Package stablenet implements the chainbench capabilities specific to the
// stablenet chain (its governance system contracts), separate from the common
// set. Importing it for side effects loads stablenet.jsonl and registers its
// handlers into the capability registry.
package stablenet

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/pkg/mcp/capability"
)

//go:embed stablenet.jsonl
var catalog []byte

func init() {
	if err := capability.LoadCatalog(catalog); err != nil {
		panic(err)
	}
	capability.RegisterHandler("v1", "stablenet", "governance.propose_mint", proposeMint)
}

func proposeMint(_ context.Context, args map[string]any) (string, error) {
	beneficiary := capability.ArgString(args, "beneficiary", "")
	if beneficiary == "" {
		return "", fmt.Errorf("beneficiary is required")
	}
	amount := capability.ArgBigInt(args, "amount")
	if amount == nil {
		return "", fmt.Errorf("amount is required (decimal wei)")
	}
	timestamp := capability.ArgBigInt(args, "timestamp")
	if timestamp == nil {
		return "", fmt.Errorf("timestamp is required (decimal)")
	}
	proof := govbind.MintProof(
		beneficiary, amount, timestamp,
		capability.ArgString(args, "deposit_id", ""),
		capability.ArgString(args, "bank_reference", ""),
		capability.ArgString(args, "memo", ""),
	)
	return govbind.ProposeMintCall(proof), nil
}
