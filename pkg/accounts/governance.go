package accounts

import (
	"encoding/hex"
	"math/big"
	"strings"
)

// This file is the go-stablenet GovBase binding: calldata builders for the
// proposal lifecycle (propose -> approve -> execute), the MintProof encoder, and
// the two decoders needed to drive it (the proposalId from a ProposalCreated log
// and the status from the proposals() getter). These are the pure pieces the
// governance write regression cases are built from; a live signer path (node-side
// eth_sendTransaction from unlocked validators, for the multi-signer quorum) is a
// separate concern, so everything here is unit-testable without a network.

// ProposalCreatedTopic is topic0 of the GovBase ProposalCreated event; its first
// indexed field (topics[1]) is the proposalId. The GovBase Solidity is not in
// this repo, so this hash is pinned from the regression suite
// (tests/regression/lib/common.sh PROPOSAL_CREATED_SIG) rather than derived from
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
	return EncodeABI(
		Address(beneficiary),
		Uint(amount),
		Uint(timestamp),
		StringArg(depositID),
		StringArg(bankReference),
		StringArg(memo),
	)
}

// ProposeMintCall builds calldata for GovMinter.proposeMint(bytes) wrapping proof
// (typically from MintProof). Returns 0x-hex.
func ProposeMintCall(proof []byte) string {
	return EncodeCallArgs("proposeMint(bytes)", Bytes(proof))
}

// ProposeAddMemberCall builds calldata for a governance
// proposeAddMember(address,uint32): add member with the new quorum. Returns 0x-hex.
func ProposeAddMemberCall(member string, newQuorum uint32) string {
	return EncodeCallArgs("proposeAddMember(address,uint32)",
		Address(member), Uint(big.NewInt(int64(newQuorum))))
}

// ApproveProposalCall builds calldata for approveProposal(uint256). A quorum-th
// approval auto-executes the proposal inside the approve tx. Returns 0x-hex.
func ApproveProposalCall(id *big.Int) string {
	return EncodeCallArgs("approveProposal(uint256)", Uint(id))
}

// ExecuteProposalCall builds calldata for executeProposal(uint256), the manual
// execute for proposals that did not auto-execute on the quorum approval.
func ExecuteProposalCall(id *big.Int) string {
	return EncodeCallArgs("executeProposal(uint256)", Uint(id))
}

// ProposalsCall builds calldata for the proposals(uint256) auto-getter, whose
// return the caller decodes with DecodeProposalStatus.
func ProposalsCall(id *big.Int) string {
	return EncodeCallArgs("proposals(uint256)", Uint(id))
}

// ExtractProposalID returns the proposalId from the ProposalCreated log among
// logs (its first indexed field) and whether one was found. It filters by topic0
// rather than taking the first log, because a payable propose emits Transfer
// first.
func ExtractProposalID(logs []Log) (*big.Int, bool) {
	log, found := FindLog(logs, ProposalCreatedTopic)
	if !found || len(log.Topics) < 2 {
		return nil, false
	}
	return WordToBig(log.Topics[1])
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
