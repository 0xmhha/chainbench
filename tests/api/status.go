// This file adds node-status read cases (ported from tests/regression/g-api g4).
//
// # Test: txpool-status
//
// Intent:   txpool_status must return parseable pending/queued counts — the read
//
//	the MCP txpool tool and operators rely on.
//
// Applies:  all chains. Requires: "rpc".
// Method:   txpool_status; assert pending and queued parse as hex numbers.
// Pass:     both counts parse (any value, including zero).
//
// # Test: chain-not-syncing
//
// Intent:   a healthy launched node reports eth_syncing == false (it is caught
//
//	up, not stuck downloading).
//
// Applies:  all chains. Requires: "rpc".
// Method:   eth_syncing; assert the result is boolean false.
// Pass:     eth_syncing is false.

package api

import (
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "txpool-status",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           txpoolStatus,
	})
	testkit.Register(testkit.Case{
		Name:         "chain-not-syncing",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           chainNotSyncing,
	})
}

func txpoolStatus(t *testkit.T) {
	var status struct {
		Pending string `json:"pending"`
		Queued  string `json:"queued"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "txpool_status", &status), "txpool_status")
	t.Truef(isHexNumber(status.Pending), "pending count %q is a hex number", status.Pending)
	t.Truef(isHexNumber(status.Queued), "queued count %q is a hex number", status.Queued)
}

func chainNotSyncing(t *testkit.T) {
	var syncing any
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_syncing", &syncing), "eth_syncing")
	b, ok := syncing.(bool)
	t.Truef(ok && !b, "eth_syncing is false (node caught up), got %v", syncing)
}

// isHexNumber reports whether s is a 0x-prefixed hex integer.
func isHexNumber(s string) bool {
	if !strings.HasPrefix(s, "0x") || len(s) < 3 {
		return false
	}
	for _, c := range s[2:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
