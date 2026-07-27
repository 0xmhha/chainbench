// # Test: node-address-returned
//
// Intent:   every node must expose its own validator-identity address over the
//
//	istanbul namespace — the basic per-node identity handle other
//	consensus queries key off (ported from tests/regression/g-api/
//	g3-01-node-address.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc".
// Method:   istanbul_nodeAddress (no params) on each node in the set.
// Pass:     each node returns a 0x-prefixed 20-byte (42-char) address.
//
// # Test: wbft-extra-info-fields
//
// Intent:   the WBFT extra header payload must carry its seal/gas-tip fields so
//
//	downstream seal-quorum checks have something to read (ported from
//	tests/regression/g-api/g3-04-get-wbft-extra.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc".
// Method:   istanbul_getWbftExtraInfo("latest"); inspect the result object.
// Pass:     the response object carries gasTip, committedSeal and preparedSeal
//
//	fields.
//
// # Test: istanbul-status-fields
//
// Intent:   the istanbul_status range report must expose its sealer-activity /
//
//	author / range / round-stats sections (ported from
//	tests/regression/g-api/g3-05-istanbul-status.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc".
// Method:   istanbul_status(start,end) over a block range; inspect the result.
// Pass:     the response carries sealerActivity, author, blockRange and
//
//	roundStats fields.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase against a live NodeSet (the sibling _test.go validates
// registration and runs each against a mock node).
package consensus

import (
	"encoding/json"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "node-address-returned",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           nodeAddressReturned,
	})
	testkit.Register(testkit.Case{
		Name:         "wbft-extra-info-fields",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           wbftExtraInfoFields,
	})
	testkit.Register(testkit.Case{
		Name:         "istanbul-status-fields",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           istanbulStatusFields,
	})
}

func nodeAddressReturned(t *testkit.T) {
	nodes := t.NodeSet().Nodes
	t.Truef(len(nodes) > 0, "node set is non-empty")
	for _, n := range nodes {
		var addr string
		t.NoErr(t.Node(n.Index).Call(t.Ctx(), "istanbul_nodeAddress", &addr), "istanbul_nodeAddress")
		t.Truef(strings.HasPrefix(addr, "0x") && len(addr) == 42,
			"node%d istanbul_nodeAddress returns a 20-byte address, got %q", n.Index, addr)
	}
}

func wbftExtraInfoFields(t *testkit.T) {
	var extra map[string]json.RawMessage
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, "latest"),
		"istanbul_getWbftExtraInfo(latest)")
	for _, field := range []string{"gasTip", "committedSeal", "preparedSeal"} {
		_, ok := extra[field]
		t.Truef(ok, "WBFTExtra has %s field", field)
	}
}

func istanbulStatusFields(t *testkit.T) {
	// A block-1..latest range is enough to exercise the report shape without
	// needing to first resolve the head height.
	var status map[string]json.RawMessage
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_status", &status, "0x1", "latest"),
		"istanbul_status(range)")
	for _, field := range []string{"sealerActivity", "author", "blockRange", "roundStats"} {
		_, ok := status[field]
		t.Truef(ok, "istanbul_status has %s field", field)
	}
}
