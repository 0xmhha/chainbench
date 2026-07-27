// # Test: commit-signers-quorum
//
// Intent:   the block's commit-signer report must name an author and gather a
//
//	committer set at or above BFT quorum — proof the block was finalized
//	by enough validators (ported from tests/regression/g-api/
//	g3-03-get-commit-signers.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc", "consensus".
// Method:   istanbul_getCommitSignersFromBlock("latest"); read Author and
//
//	Committers; quorum = ceil(2N/3) over the current validator set.
//
// Pass:     Author is a 0x address and len(Committers) >= quorum.
//
// # Test: wbft-seals-quorum
//
// Intent:   the latest block's committed and prepared seals must each be signed
//
//	by a quorum of sealers and carry a non-empty aggregate signature
//	(ported from tests/regression/b-wbft/b-02-wbft-extra-seal.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc", "consensus".
// Method:   istanbul_getWbftExtraInfo("latest"); inspect committedSeal and
//
//	preparedSeal.
//
// Pass:     each seal lists sealers >= quorum and a signature longer than the
//
//	bare "0x" prefix.
//
// # Test: prev-seals-quorum
//
// Intent:   the latest block must carry the previous block's votes: its
//
//	prevCommittedSeal and prevPreparedSeal each gather a quorum of
//	sealers (block N carries block N-1's seals; ported from
//	tests/regression/b-wbft/b-11-prev-committed-seal.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc", "consensus".
// Method:   istanbul_getWbftExtraInfo("latest"); inspect prevCommittedSeal and
//
//	prevPreparedSeal.
//
// Pass:     each prev-seal lists sealers >= quorum.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase against a live NodeSet (the sibling _test.go validates
// registration and runs each against a mock node).
package consensus

import (
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "commit-signers-quorum",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc", "consensus"},
		Fn:           commitSignersQuorum,
	})
	testkit.Register(testkit.Case{
		Name:         "wbft-seals-quorum",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc", "consensus"},
		Fn:           wbftSealsQuorum,
	})
	testkit.Register(testkit.Case{
		Name:         "prev-seals-quorum",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc", "consensus"},
		Fn:           prevSealsQuorum,
	})
}

// wbftSeal is one seal entry inside the WBFT extra payload: the set of sealer
// addresses that signed and the aggregate signature over them.
type wbftSeal struct {
	Sealers   []string `json:"sealers"`
	Signature string   `json:"signature"`
}

// wbftExtraSeals is the subset of istanbul_getWbftExtraInfo we assert on.
type wbftExtraSeals struct {
	CommittedSeal     wbftSeal `json:"committedSeal"`
	PreparedSeal      wbftSeal `json:"preparedSeal"`
	PrevCommittedSeal wbftSeal `json:"prevCommittedSeal"`
	PrevPreparedSeal  wbftSeal `json:"prevPreparedSeal"`
}

// commitSigners is the istanbul_getCommitSignersFromBlock result. JSON decoding
// is case-insensitive, so the CamelCase tags also match a lowercase server.
type commitSigners struct {
	Author     string   `json:"Author"`
	Committers []string `json:"Committers"`
}

// quorumOf returns the BFT quorum ceil(2n/3) for an n-validator set.
func quorumOf(n int) int { return (2*n + 2) / 3 }

// validatorCount reports how many validators the engine currently lists.
func validatorCount(t *testkit.T) int {
	var vals []string
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getValidators", &vals, "latest"), "istanbul_getValidators")
	return len(vals)
}

func commitSignersQuorum(t *testkit.T) {
	quorum := quorumOf(validatorCount(t))
	var cs commitSigners
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getCommitSignersFromBlock", &cs, "latest"),
		"istanbul_getCommitSignersFromBlock(latest)")
	t.Truef(strings.HasPrefix(cs.Author, "0x") && len(cs.Author) == 42,
		"Author is a 20-byte address, got %q", cs.Author)
	t.Truef(len(cs.Committers) >= quorum,
		"committers >= quorum (%d), got %d", quorum, len(cs.Committers))
}

func wbftSealsQuorum(t *testkit.T) {
	quorum := quorumOf(validatorCount(t))
	var extra wbftExtraSeals
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, "latest"),
		"istanbul_getWbftExtraInfo(latest)")

	t.Truef(len(extra.CommittedSeal.Sealers) >= quorum,
		"committedSeal.sealers >= quorum (%d), got %d", quorum, len(extra.CommittedSeal.Sealers))
	t.Truef(len(extra.CommittedSeal.Signature) > 2,
		"committedSeal.signature is non-empty, got %q", extra.CommittedSeal.Signature)

	t.Truef(len(extra.PreparedSeal.Sealers) >= quorum,
		"preparedSeal.sealers >= quorum (%d), got %d", quorum, len(extra.PreparedSeal.Sealers))
	t.Truef(len(extra.PreparedSeal.Signature) > 2,
		"preparedSeal.signature is non-empty, got %q", extra.PreparedSeal.Signature)
}

func prevSealsQuorum(t *testkit.T) {
	quorum := quorumOf(validatorCount(t))
	var extra wbftExtraSeals
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, "latest"),
		"istanbul_getWbftExtraInfo(latest)")

	t.Truef(len(extra.PrevCommittedSeal.Sealers) >= quorum,
		"prevCommittedSeal.sealers >= quorum (%d), got %d", quorum, len(extra.PrevCommittedSeal.Sealers))
	t.Truef(len(extra.PrevPreparedSeal.Sealers) >= quorum,
		"prevPreparedSeal.sealers >= quorum (%d), got %d", quorum, len(extra.PrevPreparedSeal.Sealers))
}
