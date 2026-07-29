// Package govbind is the go-stablenet GovBase binding: calldata builders for the
// proposal lifecycle (propose -> approve -> execute), the MintProof encoder, and
// the two decoders needed to drive it (the proposalId from a ProposalCreated log
// and the status from the proposals() getter). This is stablenet-specific chain
// knowledge, so it lives under pkg/chains/stablenet — the generic pkg/accounts
// package holds only the chain-agnostic ABI/event/tx helpers these build on.
// A live signer path (node-side eth_sendTransaction from unlocked validators,
// for the multi-signer quorum) is a separate concern, so everything here is
// pure and unit-testable without a network.
package govbind

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/accounts"
)

// ProposalCreatedTopic is topic0 of the GovBase ProposalCreated event; its first
// indexed field (topics[1]) is the proposalId. The GovBase Solidity is not in
// this repo, so this hash is pinned from the go-stablenet regression suite
// (lib/common.sh PROPOSAL_CREATED_SIG) rather than derived from
// a signature string. Filtering on it is required because proposeBurn (payable)
// emits Transfer before ProposalCreated, so the first log is not the proposal.
const ProposalCreatedTopic = "0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"

// Proposal status values from GovBase: the states a proposal moves through, as
// reported by the proposals() getter's 10th field (a uint8).
const (
	ProposalNone      uint8 = 0
	ProposalVoting    uint8 = 1
	ProposalApproved  uint8 = 2
	ProposalExecuted  uint8 = 3
	ProposalCancelled uint8 = 4
	ProposalExpired   uint8 = 5
	ProposalFailed    uint8 = 6
	ProposalRejected  uint8 = 7
)

// MintProof ABI-encodes a GovMinter MintProof — abi.encode(beneficiary, amount,
// timestamp, depositID, bankReference, memo) — the proofData bytes passed to
// proposeMint(bytes). Mirrors the regression suite's eth_abi encoding.
func MintProof(beneficiary string, amount, timestamp *big.Int, depositID, bankReference, memo string) []byte {
	return accounts.EncodeABI(
		accounts.Address(beneficiary),
		accounts.Uint(amount),
		accounts.Uint(timestamp),
		accounts.StringArg(depositID),
		accounts.StringArg(bankReference),
		accounts.StringArg(memo),
	)
}

// ProposeMintCall builds calldata for GovMinter.proposeMint(bytes) wrapping proof
// (typically from MintProof). Returns 0x-hex.
func ProposeMintCall(proof []byte) string {
	return accounts.EncodeCallArgs("proposeMint(bytes)", accounts.Bytes(proof))
}

// BurnProof ABI-encodes a GovMinter BurnProof — abi.encode(from, amount,
// timestamp, withdrawalID, referenceID, memo) — the proofData bytes passed to
// proposeBurn(bytes). Same layout as MintProof with `from` (the burn target) in
// place of the beneficiary.
func BurnProof(from string, amount, timestamp *big.Int, withdrawalID, referenceID, memo string) []byte {
	return accounts.EncodeABI(
		accounts.Address(from),
		accounts.Uint(amount),
		accounts.Uint(timestamp),
		accounts.StringArg(withdrawalID),
		accounts.StringArg(referenceID),
		accounts.StringArg(memo),
	)
}

// ProposeBurnCall builds calldata for GovMinter.proposeBurn(bytes). The call is
// payable: msg.value must equal the burn amount, and the proposer must be the
// proof's `from`. Returns 0x-hex.
func ProposeBurnCall(proof []byte) string {
	return accounts.EncodeCallArgs("proposeBurn(bytes)", accounts.Bytes(proof))
}

// CancelProposalCall builds calldata for cancelProposal(uint256): the proposer
// cancels the proposal. For a burn proposal, cancelling makes the burned value
// refundable to the proposer. Returns 0x-hex.
func CancelProposalCall(id *big.Int) string {
	return accounts.EncodeCallArgs("cancelProposal(uint256)", accounts.Uint(id))
}

// ClaimBurnRefundCall builds calldata for claimBurnRefund() — the caller claims
// its refundable burn balance into its native balance. No arguments. Returns
// 0x-hex.
func ClaimBurnRefundCall() string {
	return accounts.EncodeCallArgs("claimBurnRefund()")
}

// DisapproveProposalCall builds calldata for disapproveProposal(uint256): a
// disapproval vote. A quorum of disapprovals rejects the proposal (making a
// burn's value refundable). Returns 0x-hex.
func DisapproveProposalCall(id *big.Int) string {
	return accounts.EncodeCallArgs("disapproveProposal(uint256)", accounts.Uint(id))
}

// Burn-refund event topics (GovMinter, Boho v2). Pinned from the go-stablenet
// regression suite (lib/common.sh) since the GovMinter Solidity is not in this
// repo. BurnRefundClaimed also derives from EventTopic("BurnRefundClaimed(address,uint256)").
const (
	BurnRefundClaimedTopic   = "0x9543fa265d2616af3e7021d8b5a7d1271eb7bba960908675ce3bddaf60c1af24"
	BurnDepositRefundedTopic = "0x334fe3eaa506b12e7e46ba469c310822737a959f2553b3cb38dff68085291aed"
)

// ProposeAddMemberCall builds calldata for a governance
// proposeAddMember(address,uint32): add member with the new quorum. Returns 0x-hex.
func ProposeAddMemberCall(member string, newQuorum uint32) string {
	return accounts.EncodeCallArgs("proposeAddMember(address,uint32)",
		accounts.Address(member), accounts.Uint(big.NewInt(int64(newQuorum))))
}

// ApproveProposalCall builds calldata for approveProposal(uint256). A quorum-th
// approval auto-executes the proposal inside the approve tx. Returns 0x-hex.
func ApproveProposalCall(id *big.Int) string {
	return accounts.EncodeCallArgs("approveProposal(uint256)", accounts.Uint(id))
}

// ExecuteProposalCall builds calldata for executeProposal(uint256), the manual
// execute for proposals that did not auto-execute on the quorum approval.
func ExecuteProposalCall(id *big.Int) string {
	return accounts.EncodeCallArgs("executeProposal(uint256)", accounts.Uint(id))
}

// ProposalsCall builds calldata for the proposals(uint256) auto-getter, whose
// return the caller decodes with DecodeProposalStatus.
func ProposalsCall(id *big.Int) string {
	return accounts.EncodeCallArgs("proposals(uint256)", accounts.Uint(id))
}

// ExtractProposalID returns the proposalId from the ProposalCreated log among
// logs (its first indexed field) and whether one was found. It filters by topic0
// rather than taking the first log, because a payable propose emits Transfer
// first.
func ExtractProposalID(logs []accounts.Log) (*big.Int, bool) {
	log, found := accounts.FindLog(logs, ProposalCreatedTopic)
	if !found || len(log.Topics) < 2 {
		return nil, false
	}
	return accounts.WordToBig(log.Topics[1])
}

// DecodeProposalStatus decodes the status (10th field, uint8) from a proposals()
// getter return. The proposals() tuple is 10 static fields
// (bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8),
// so the status is the last byte of the word at index 9. Returns (0, false) on a
// short/malformed return.
func DecodeProposalStatus(hexRet string) (uint8, bool) {
	b, err := hex.DecodeString(strings.TrimPrefix(hexRet, "0x"))
	if err != nil || len(b) < 10*32 {
		return 0, false
	}
	return b[10*32-1], true
}
