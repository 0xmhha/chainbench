package accounts_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/accounts"
)

// canonical vectors from the regression suite's Python eth_abi encoding
// (tests/regression/f-system-contracts/f2-01-mint-proposal.sh):
//
//	proof = encode(['address','uint256','uint256','string','string','string'],
//	               ['0x..ab', 1e18, 1700000000, 'DEP-1', 'BANK-1', 'memo'])
//	call  = keccak('proposeMint(bytes)')[:4] + encode(['bytes'], [proof])
const (
	wantProof = "00000000000000000000000000000000000000000000000000000000000000ab" + // beneficiary
		"0000000000000000000000000000000000000000000000000de0b6b3a7640000" + // amount 1e18
		"000000000000000000000000000000000000000000000000000000006553f100" + // timestamp
		"00000000000000000000000000000000000000000000000000000000000000c0" + // offset depositID
		"0000000000000000000000000000000000000000000000000000000000000100" + // offset bankRef
		"0000000000000000000000000000000000000000000000000000000000000140" + // offset memo
		"0000000000000000000000000000000000000000000000000000000000000005" + // len "DEP-1"
		"4445502d31000000000000000000000000000000000000000000000000000000" + // "DEP-1"
		"0000000000000000000000000000000000000000000000000000000000000006" + // len "BANK-1"
		"42414e4b2d310000000000000000000000000000000000000000000000000000" + // "BANK-1"
		"0000000000000000000000000000000000000000000000000000000000000004" + // len "memo"
		"6d656d6f00000000000000000000000000000000000000000000000000000000" //  "memo"
	wantProposeMintCall = "0x1e5e0426" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000180"
)

func TestMintProofAndProposeMintCall(t *testing.T) {
	proof := accounts.MintProof("0x00000000000000000000000000000000000000ab",
		big.NewInt(1000000000000000000), big.NewInt(1700000000), "DEP-1", "BANK-1", "memo")
	if got := hexString(proof); got != wantProof {
		t.Errorf("MintProof:\n got=%s\nwant=%s", got, wantProof)
	}
	// the proposeMint(bytes) call: selector + [offset 0x20][length 0x180] + proof.
	call := accounts.ProposeMintCall(proof)
	if !strings.HasPrefix(call, wantProposeMintCall) {
		t.Errorf("ProposeMintCall head:\n got=%s\nwant prefix %s", call, wantProposeMintCall)
	}
	if want := wantProposeMintCall + wantProof; call != want {
		t.Errorf("ProposeMintCall full:\n got=%s\nwant=%s", call, want)
	}
}

func TestGovernanceCalldataBuilders(t *testing.T) {
	if got := accounts.ProposeAddMemberCall("0x00000000000000000000000000000000000000ab", 3); got !=
		"0x"+sel4("proposeAddMember(address,uint32)")+
			z64[:24]+"00000000000000000000000000000000000000ab"+z64[:63]+"3" {
		t.Errorf("ProposeAddMemberCall = %s", got)
	}
	if got := accounts.ApproveProposalCall(big.NewInt(7)); got != "0x"+sel4("approveProposal(uint256)")+z64[:63]+"7" {
		t.Errorf("ApproveProposalCall = %s", got)
	}
	if got := accounts.ExecuteProposalCall(big.NewInt(7)); got != "0x"+sel4("executeProposal(uint256)")+z64[:63]+"7" {
		t.Errorf("ExecuteProposalCall = %s", got)
	}
	if got := accounts.ProposalsCall(big.NewInt(7)); got != "0x"+sel4("proposals(uint256)")+z64[:63]+"7" {
		t.Errorf("ProposalsCall = %s", got)
	}
}

func TestExtractProposalID(t *testing.T) {
	logs := []accounts.Log{
		// a Transfer log first (payable propose emits it before ProposalCreated).
		{Topics: []string{accounts.EventTopic("Transfer(address,address,uint256)"), "0x00", "0x00"}},
		{Topics: []string{accounts.ProposalCreatedTopic, "0x" + z64[:63] + "5"}},
	}
	id, ok := accounts.ExtractProposalID(logs)
	if !ok || id.Int64() != 5 {
		t.Fatalf("ExtractProposalID = %v,%v want 5,true", id, ok)
	}
	// no ProposalCreated log -> not found.
	if _, ok := accounts.ExtractProposalID(logs[:1]); ok {
		t.Errorf("ExtractProposalID found an id with no ProposalCreated log")
	}
	// ProposalCreated with only topic0 (no indexed id) -> not found.
	bad := []accounts.Log{{Topics: []string{accounts.ProposalCreatedTopic}}}
	if _, ok := accounts.ExtractProposalID(bad); ok {
		t.Errorf("ExtractProposalID accepted a log with no indexed proposalId")
	}
}

func TestDecodeProposalStatus(t *testing.T) {
	// proposals() tuple: 10 static words; status is the last byte of word[9].
	tuple := strings.Repeat(z64, 9) + z64[:63] + "3" // status = Executed (3)
	st, ok := accounts.DecodeProposalStatus("0x" + tuple)
	if !ok || st != accounts.ProposalExecuted {
		t.Errorf("DecodeProposalStatus = %d,%v want 3,true", st, ok)
	}
	// short return -> not decodable.
	if _, ok := accounts.DecodeProposalStatus("0x1234"); ok {
		t.Errorf("DecodeProposalStatus accepted a short return")
	}
	if _, ok := accounts.DecodeProposalStatus("0x"); ok {
		t.Errorf("DecodeProposalStatus accepted an empty return")
	}
}
