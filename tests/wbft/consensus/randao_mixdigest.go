// This file ports wemix4 WBFT-010 (RandaoReveal / MixDigest): WBFT blocks carry
// a RandaoReveal in their extra payload and the resulting randao mix is surfaced
// as the block header's mixHash (MixDigest). Both must be present and non-zero
// on a produced block.
//
// # Test: randao-and-mixdigest-present
//
// Intent:   a WBFT block exposes both randomness fields — istanbul_getWbftExtraInfo
//
//	carries a non-empty randaoReveal, and eth_getBlockByNumber carries a
//	non-zero mixHash (the derived MixDigest).
//
// Applies:  wbft. Requires: the "rpc" capability.
// Method:   read the head block: istanbul_getWbftExtraInfo(head).randaoReveal and
//
//	eth_getBlockByNumber(head).mixHash.
//
// Pass:     randaoReveal is a non-empty hex string and mixHash is present and not
//
//	the all-zero hash.
//
// This is chainbench TEST CODE (requirement #16): it reads a live node, so it is
// only meaningful against a running network (the sibling _test.go validates
// registration and the pass/fail decision against a mock).
package consensus

import (
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/testkit"
)

const zeroHash = "0x0000000000000000000000000000000000000000000000000000000000000000"

func init() {
	testkit.Register(testkit.Case{
		Name:         "randao-and-mixdigest-present",
		Category:     "consensus",
		ChainCompat:  []string{"wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           randaoAndMixDigestPresent,
	})
}

// hexNonEmpty reports whether s is a hex string with at least one nibble of
// payload (i.e. present and not "" / "0x").
func hexNonEmpty(s string) bool {
	s = strings.TrimSpace(s)
	return s != "" && s != "0x"
}

func randaoAndMixDigestPresent(t *testkit.T) {
	head, err := t.Primary().BlockNumber(t.Ctx())
	t.NoErr(err, "eth_blockNumber")
	blockHex := fmt.Sprintf("0x%x", head)

	// RandaoReveal lives in the WBFT extra payload.
	var extra struct {
		RandaoReveal string `json:"randaoReveal"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, blockHex),
		"istanbul_getWbftExtraInfo(head)")
	t.Truef(hexNonEmpty(extra.RandaoReveal), "randaoReveal present at block %d (got %q)", head, extra.RandaoReveal)

	// The derived randao mix is the header's mixHash (MixDigest).
	var block struct {
		MixHash string `json:"mixHash"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, blockHex, false),
		"eth_getBlockByNumber(head)")
	t.Truef(hexNonEmpty(block.MixHash), "mixHash present at block %d (got %q)", head, block.MixHash)
	t.Truef(block.MixHash != zeroHash, "mixHash must be non-zero at block %d", head)
}
