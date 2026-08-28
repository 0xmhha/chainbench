// This file ports the PORTABLE (post-fork state) subset of the go-stablenet
// h-hardfork regression suite (regression/h-hardfork). chainbench's
// stablenet network activates the Anzeon/Boho hardforks at genesis (block 0), so
// the post-fork artifacts these cases observe — the P-256 precompile, the
// GovMinter v2 bytecode, and the anzeon chain config — are already present on a
// running stablenet and can be checked as ordinary eth_call / eth_getCode /
// eth_getBlockByNumber reads.
//
// # Test: p256-precompile-active
//
// Intent:   the secp256r1 (P-256, RIP-7212) precompile at 0x00..0100 is active
//
//	after Boho and verifies a valid NIST P-256 signature, returning the
//	success word (0x..01).
//
// Applies:  stablenet. Requires: "rpc".
// Method:   eth_call the precompile with a valid 160-byte hash||r||s||x||y test
//
//	vector; assert the return equals the 32-byte one-word success value.
//
// Pass:     the precompile returns 0x..01 (signature valid).
//
// # Test: p256-rejects-invalid
//
// Intent:   the P-256 precompile does NOT return success for a corrupted
//
//	signature (regression h-17) nor for a truncated <160-byte input
//	(regression h-18).
//
// Applies:  stablenet. Requires: "rpc".
// Method:   eth_call with a corrupted-r input and with a 64-byte short input;
//
//	assert neither returns the success word (an RPC error is acceptable —
//	it also means "not verified").
//
// Pass:     neither call returns 0x..01.
package anzeon

import (
	"github.com/0xmhha/chainbench/internal/testkit"
)

// p256Precompile is the secp256r1 (RIP-7212) precompile address activated by Boho.
const p256Precompile = "0x0000000000000000000000000000000000000100"

// p256SuccessWord is the one-word result returned for a verified P-256 signature.
const p256SuccessWord = "0x0000000000000000000000000000000000000000000000000000000000000001"

// p256ValidInput is a valid 160-byte NIST P-256 test vector (hash||r||s||x||y),
// from regression h-02/h-19.
const p256ValidInput = "0x" +
	"bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023" + // hash
	"2927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838" + // r
	"c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e" + // s
	"04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0" + // x
	"f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70" //   y

// p256InvalidSigInput corrupts the first byte of r (29->ff), so the signature is
// no longer valid (regression h-17).
const p256InvalidSigInput = "0x" +
	"bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023" +
	"ff27b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838" +
	"c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e" +
	"04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0" +
	"f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70"

// p256ShortInput is only 64 bytes (hash||r), missing s, x, y (regression h-18).
const p256ShortInput = "0x" +
	"bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023" +
	"2927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838"

func init() {
	testkit.Register(testkit.Case{
		Name:         "p256-precompile-active",
		Category:     "hardfork",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           p256PrecompileActive,
	})
	testkit.Register(testkit.Case{
		Name:         "p256-rejects-invalid",
		Category:     "hardfork",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           p256RejectsInvalid,
	})
}

func p256PrecompileActive(t *testkit.T) {
	var result string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_call", &result,
		map[string]string{"to": p256Precompile, "data": p256ValidInput}, "latest"),
		"eth_call P-256 (valid vector)")
	t.Equalf(result, p256SuccessWord, "P-256 verifies the valid signature (returns 0x..01)")
}

func p256RejectsInvalid(t *testkit.T) {
	// A corrupted signature must not verify; an RPC error is also acceptable.
	var got string
	if err := t.Primary().Call(t.Ctx(), "eth_call", &got,
		map[string]string{"to": p256Precompile, "data": p256InvalidSigInput}, "latest"); err == nil {
		t.Truef(got != p256SuccessWord, "P-256 must not return success for a corrupted signature (got %s)", got)
	}
	// A short (<160-byte) input must not verify either.
	got = ""
	if err := t.Primary().Call(t.Ctx(), "eth_call", &got,
		map[string]string{"to": p256Precompile, "data": p256ShortInput}, "latest"); err == nil {
		t.Truef(got != p256SuccessWord, "P-256 must not return success for a 64-byte short input (got %s)", got)
	}
}
