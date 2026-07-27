package govbind_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
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
	proof := govbind.MintProof("0x00000000000000000000000000000000000000ab",
		big.NewInt(1000000000000000000), big.NewInt(1700000000), "DEP-1", "BANK-1", "memo")
	if got := hexString(proof); got != wantProof {
		t.Errorf("MintProof:\n got=%s\nwant=%s", got, wantProof)
	}
	// the proposeMint(bytes) call: selector + [offset 0x20][length 0x180] + proof.
	call := govbind.ProposeMintCall(proof)
	if !strings.HasPrefix(call, wantProposeMintCall) {
		t.Errorf("ProposeMintCall head:\n got=%s\nwant prefix %s", call, wantProposeMintCall)
	}
	if want := wantProposeMintCall + wantProof; call != want {
		t.Errorf("ProposeMintCall full:\n got=%s\nwant=%s", call, want)
	}
}

func TestBurnProofAndProposeBurnCall(t *testing.T) {
	// BurnProof has the same ABI layout as MintProof, so the same inputs encode
	// identically; the calldata differs only in the proposeBurn selector.
	burn := govbind.BurnProof("0x00000000000000000000000000000000000000ab",
		big.NewInt(1000000000000000000), big.NewInt(1700000000), "DEP-1", "BANK-1", "memo")
	if got := hexString(burn); got != wantProof {
		t.Errorf("BurnProof:\n got=%s\nwant=%s", got, wantProof)
	}
	call := govbind.ProposeBurnCall(burn)
	wantHead := "0x" + sel4("proposeBurn(bytes)") +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000180"
	if want := wantHead + wantProof; call != want {
		t.Errorf("ProposeBurnCall:\n got=%s\nwant=%s", call, want)
	}
}

func TestBurnRefundBuilders(t *testing.T) {
	if got := govbind.CancelProposalCall(big.NewInt(7)); got != "0x"+sel4("cancelProposal(uint256)")+z64[:63]+"7" {
		t.Errorf("CancelProposalCall = %s", got)
	}
	if got := govbind.ClaimBurnRefundCall(); got != "0x"+sel4("claimBurnRefund()") {
		t.Errorf("ClaimBurnRefundCall = %s (want bare selector)", got)
	}
	// BurnRefundClaimed derives from its signature; BurnDepositRefunded is pinned.
	if govbind.BurnRefundClaimedTopic != accounts.EventTopic("BurnRefundClaimed(address,uint256)") {
		t.Errorf("BurnRefundClaimedTopic does not match the derived topic")
	}
	if len(govbind.BurnDepositRefundedTopic) != 66 {
		t.Errorf("BurnDepositRefundedTopic malformed: %q", govbind.BurnDepositRefundedTopic)
	}
}

func TestGovernanceCalldataBuilders(t *testing.T) {
	if got := govbind.ProposeAddMemberCall("0x00000000000000000000000000000000000000ab", 3); got !=
		"0x"+sel4("proposeAddMember(address,uint32)")+
			z64[:24]+"00000000000000000000000000000000000000ab"+z64[:63]+"3" {
		t.Errorf("ProposeAddMemberCall = %s", got)
	}
	if got := govbind.ApproveProposalCall(big.NewInt(7)); got != "0x"+sel4("approveProposal(uint256)")+z64[:63]+"7" {
		t.Errorf("ApproveProposalCall = %s", got)
	}
	if got := govbind.ExecuteProposalCall(big.NewInt(7)); got != "0x"+sel4("executeProposal(uint256)")+z64[:63]+"7" {
		t.Errorf("ExecuteProposalCall = %s", got)
	}
	if got := govbind.ProposalsCall(big.NewInt(7)); got != "0x"+sel4("proposals(uint256)")+z64[:63]+"7" {
		t.Errorf("ProposalsCall = %s", got)
	}
}

func TestExtractProposalID(t *testing.T) {
	logs := []accounts.Log{
		// a Transfer log first (payable propose emits it before ProposalCreated).
		{Topics: []string{accounts.EventTopic("Transfer(address,address,uint256)"), "0x00", "0x00"}},
		{Topics: []string{govbind.ProposalCreatedTopic, "0x" + z64[:63] + "5"}},
	}
	id, ok := govbind.ExtractProposalID(logs)
	if !ok || id.Int64() != 5 {
		t.Fatalf("ExtractProposalID = %v,%v want 5,true", id, ok)
	}
	// no ProposalCreated log -> not found.
	if _, ok := govbind.ExtractProposalID(logs[:1]); ok {
		t.Errorf("ExtractProposalID found an id with no ProposalCreated log")
	}
	// ProposalCreated with only topic0 (no indexed id) -> not found.
	bad := []accounts.Log{{Topics: []string{govbind.ProposalCreatedTopic}}}
	if _, ok := govbind.ExtractProposalID(bad); ok {
		t.Errorf("ExtractProposalID accepted a log with no indexed proposalId")
	}
}

func TestDecodeProposalStatus(t *testing.T) {
	// proposals() tuple: 10 static words; status is the last byte of word[9].
	tuple := strings.Repeat(z64, 9) + z64[:63] + "3" // status = Executed (3)
	st, ok := govbind.DecodeProposalStatus("0x" + tuple)
	if !ok || st != govbind.ProposalExecuted {
		t.Errorf("DecodeProposalStatus = %d,%v want 3,true", st, ok)
	}
	// short return -> not decodable.
	if _, ok := govbind.DecodeProposalStatus("0x1234"); ok {
		t.Errorf("DecodeProposalStatus accepted a short return")
	}
	if _, ok := govbind.DecodeProposalStatus("0x"); ok {
		t.Errorf("DecodeProposalStatus accepted an empty return")
	}
}

// z64 is 64 hex zeros (a zero 32-byte word).
const z64 = "0000000000000000000000000000000000000000000000000000000000000000"

// sel4 is the 4-byte selector of sig as hex without the 0x prefix.
func sel4(sig string) string { return strings.TrimPrefix(accounts.Selector(sig), "0x") }

func hexString(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, h[c>>4], h[c&0xf])
	}
	return string(out)
}
