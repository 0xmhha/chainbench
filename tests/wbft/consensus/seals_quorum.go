// # Test: prev-seals-quorum
//
// Intent:   the latest block must carry the previous block's votes: its
//
//	prevCommittedSeal and prevPreparedSeal each gather a quorum of
//	sealers (block N carries block N-1's seals; ported from
//	regression/b-wbft/b-11-prev-committed-seal.sh).
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
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
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

// quorumOf returns the BFT quorum ceil(2n/3) for an n-validator set.
func quorumOf(n int) int { return (2*n + 2) / 3 }

// validatorCount reports how many validators the engine currently lists.
func validatorCount(t *testkit.T) int {
	var vals []string
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getValidators", &vals, "latest"), "istanbul_getValidators")
	return len(vals)
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
