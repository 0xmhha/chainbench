// This file ports the delayed-Boho fork-transition cases (from regression
// h-hardfork h-15, h-16, h-29): on a network launched with Boho delayed to block
// N (genesis.overrides.bohoBlock=N, which advertises the "delayed-boho"
// capability), the pre-fork state at an early block differs from the post-fork
// state at latest. Both cases are N-agnostic: they capture the pre-fork read at
// block 1 and wait for the latest read to cross over, so they need no knowledge
// of the activation block.
//
// # Test: govminter-code-changes-at-boho
//
// Intent:   the GovMinter system contract holds v1 bytecode before Boho and v2
//
//	after, so its code at block 1 differs from its code at latest once the
//	delayed fork activates.
//
// Applies:  stablenet. Requires "rpc" and "delayed-boho".
// Method:   read eth_getCode(GovMinter, 0x1) (pre-Boho v1); wait until
//
//	eth_getCode(GovMinter, latest) is non-empty and differs (the v1→v2 swap).
//
// Pass:     the latest code differs from the block-1 code.
//
// # Test: p256-inactive-before-boho
//
// Intent:   the P-256 precompile is installed by Boho, so a call at block 1
//
//	(pre-Boho) does not verify while the same call at latest does after the
//	delayed activation.
//
// Applies:  stablenet. Requires "rpc" and "delayed-boho".
// Method:   eth_call the P-256 precompile with a valid vector at block 1 (must
//
//	not return the success word); wait until the same call at latest returns it.
//
// Pass:     inactive at block 1, active at latest.
//
// # Test: anzeon-active-before-boho
//
// Intent:   Anzeon activates at genesis independently of the delayed Boho, so
//
//	the GovValidator system contract already holds code at block 1 (pre-Boho).
//
// Applies:  stablenet. Requires "rpc" and "delayed-boho".
// Method:   read eth_getCode(GovValidator, 0x1); expect substantial code.
// Pass:     GovValidator has system-contract code at block 1.
//
// # Test: prealloc-preserved-across-boho
//
// Intent:   the delayed Boho runtime upgrade (decodePrealloc restoration) must
//
//	not wipe genesis alloc state, so an unspent alloc account keeps its funded
//	balance and zero nonce across the fork.
//
// Applies:  stablenet. Requires "rpc" and "delayed-boho".
// Method:   read the alloc account balance at block 1; cross the fork; assert
//
//	its latest balance equals the block-1 balance and its nonce is still 0.
//
// Pass:     balance preserved and nonce == 0 after the fork.
//
// These are chainbench TEST CODE (requirement #16): they need a live network
// launched with a delayed fork, so the sibling _test.go validates registration
// and the capability gating (they skip on a normal Boho-at-genesis network).
package anzeon

import (
	"time"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "govminter-code-changes-at-boho",
		Category:     "hardfork",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc", "delayed-boho"},
		Fn:           govminterCodeChangesAtBoho,
	})
	testkit.Register(testkit.Case{
		Name:         "p256-inactive-before-boho",
		Category:     "hardfork",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc", "delayed-boho"},
		Fn:           p256InactiveBeforeBoho,
	})
	testkit.Register(testkit.Case{
		Name:         "anzeon-active-before-boho",
		Category:     "hardfork",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc", "delayed-boho"},
		Fn:           anzeonActiveBeforeBoho,
	})
	testkit.Register(testkit.Case{
		Name:         "prealloc-preserved-across-boho",
		Category:     "hardfork",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc", "delayed-boho"},
		Fn:           preallocPreservedAcrossBoho,
	})
}

// preallocUnspent is a genesis-alloc account in the stablenet preset that no
// test case ever sends from, so its funded balance and zero nonce are stable
// across the delayed fork — a spend-immune probe for decodePrealloc state
// preservation.
const preallocUnspent = "0x71562b71999873db5b286df957af199ec94617f7"

// waitForBohoCrossover captures the pre-Boho GovMinter code (block 1) and waits
// until the latest GovMinter code differs from it — the delayed v1→v2 swap —
// returning both. It fails the test if the crossover does not happen in time.
func waitForBohoCrossover(t *testkit.T) (early, latest string) {
	t.WaitFor(func() bool {
		return t.Primary().Call(t.Ctx(), "eth_getCode", &early, govMinter, "0x1") == nil &&
			early != "" && early != "0x"
	}, 60*time.Second, time.Second, "GovMinter code at block 1 (pre-Boho v1)")
	t.WaitFor(func() bool {
		return t.Primary().Call(t.Ctx(), "eth_getCode", &latest, govMinter, "latest") == nil &&
			latest != "" && latest != "0x" && latest != early
	}, 120*time.Second, 2*time.Second, "GovMinter code changes across the delayed Boho fork")
	return early, latest
}

func anzeonActiveBeforeBoho(t *testkit.T) {
	// Anzeon activates at genesis independently of the delayed Boho: the
	// GovValidator system contract already holds code at block 1 (pre-Boho).
	var code string
	t.WaitFor(func() bool {
		return t.Primary().Call(t.Ctx(), "eth_getCode", &code, govValidator, "0x1") == nil &&
			code != "" && code != "0x"
	}, 60*time.Second, time.Second, "GovValidator code at block 1 (Anzeon active pre-Boho)")
	t.Truef(len(code) > 100, "GovValidator carries substantial Anzeon system-contract code at block 1 (got %d hex chars)", len(code))
}

func preallocPreservedAcrossBoho(t *testkit.T) {
	// Balance present at block 1 (genesis alloc).
	var balEarly string
	t.WaitFor(func() bool {
		return t.Primary().Call(t.Ctx(), "eth_getBalance", &balEarly, preallocUnspent, "0x1") == nil &&
			balEarly != "" && balEarly != "0x0"
	}, 60*time.Second, time.Second, "prealloc account funded at block 1")

	// Cross the delayed Boho, then confirm the alloc state survived the runtime
	// upgrade (decodePrealloc restoration must not wipe it).
	waitForBohoCrossover(t)

	var balLatest, nonceLatest string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBalance", &balLatest, preallocUnspent, "latest"), "eth_getBalance latest")
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getTransactionCount", &nonceLatest, preallocUnspent, "latest"), "eth_getTransactionCount latest")
	t.Truef(balLatest == balEarly, "prealloc balance preserved across Boho (block1=%s latest=%s)", balEarly, balLatest)
	t.Truef(nonceLatest == "0x0", "prealloc account nonce still 0 after Boho (got %s)", nonceLatest)
}

func govminterCodeChangesAtBoho(t *testkit.T) {
	early, latest := waitForBohoCrossover(t)
	t.Truef(latest != early, "post-Boho v2 code differs from pre-Boho v1 (early=%d latest=%d hex chars)",
		len(early), len(latest))
}

func p256InactiveBeforeBoho(t *testkit.T) {
	// Before Boho the P-256 precompile is not installed: a call at block 1 does
	// not return the success word (an RPC error is also acceptable — a missing
	// precompile behaves as an ordinary account).
	var early string
	_ = t.Primary().Call(t.Ctx(), "eth_call", &early,
		map[string]string{"to": p256Precompile, "data": p256ValidInput}, "0x1")
	t.Truef(early != p256SuccessWord, "P-256 must be inactive at block 1 (pre-Boho), got %q", early)

	// After the delayed activation the same call verifies (returns 0x..01).
	var latest string
	t.WaitFor(func() bool {
		return t.Primary().Call(t.Ctx(), "eth_call", &latest,
			map[string]string{"to": p256Precompile, "data": p256ValidInput}, "latest") == nil &&
			latest == p256SuccessWord
	}, 120*time.Second, 2*time.Second, "P-256 precompile active after delayed Boho")
}
