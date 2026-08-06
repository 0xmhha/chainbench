// This file ports the secp256r1 (RIP-7212 P256VERIFY) precompile cases from the
// wemix4 suite (TX-009 valid, TX-019 invalid, TX-020 short/empty input). The
// precompile lives at 0x00..0100 and verifies a NIST P-256 ECDSA signature over
// a 160-byte input: hash(32) || r(32) || s(32) || pubX(32) || pubY(32). It
// returns the 32-byte word 0x..01 on a valid signature and empty output on any
// failure (bad signature, wrong length).
//
// # Test: secp256r1-precompile-valid
//
// Intent:   a valid P-256 signature verifies — eth_call to the precompile
//
//	returns the 32-byte word ending in 0x..01.
//
// Applies:  wbft. Requires: the "rpc" capability.
// Method:   eth_call(to=0x..0100, data=<160-byte valid vector>).
// Pass:     the result equals 0x00..01.
//
// # Test: secp256r1-precompile-invalid
//
// Intent:   an invalid signature (r=s=0 over a real pubkey) does NOT verify —
//
//	the precompile returns empty output (or a zero word), never 0x..01.
//
// Applies:  wbft. Requires: the "rpc" capability.
// Method:   eth_call(to=0x..0100, data=<160-byte input with zero r,s>).
// Pass:     the result is empty or 0x00..00 — anything but 0x..01.
//
// # Test: secp256r1-precompile-short-input
//
// Intent:   inputs that are not exactly 160 bytes are rejected without a match —
//
//	both a 159-byte input and an empty input return empty output.
//
// Applies:  wbft. Requires: the "rpc" capability.
// Method:   eth_call(to=0x..0100, data=<159 bytes>) and data=0x.
// Pass:     both calls return empty output.
//
// These are chainbench TEST CODE (requirement #16): they drive a live node's
// eth_call, so they are only meaningful against a running network (the sibling
// _test.go validates registration and the pass/fail decision against a mock).
package accounts

import (
	"strings"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// p256VerifyAddr is the RIP-7212 secp256r1 verification precompile address.
const p256VerifyAddr = "0x0000000000000000000000000000000000000100"

// p256VerifyOne is the precompile's success return: a 32-byte word == 1.
const p256VerifyOne = "0x0000000000000000000000000000000000000000000000000000000000000001"

// p256ValidInput is a 160-byte valid P-256 vector (hash||r||s||pubX||pubY) whose
// signature verifies. hash = sha256("wemix4 p256verify test").
const p256ValidInput = "0x" +
	"4fc05e7c6fa9dcfc4d09b26e3352487de9f248699f930af99fc652c7374dc89c" + // hash
	"c1e7730e0dd29f6e231649b948bf9dfa73b48f94e904615c3a6573fe05c3b6bf" + // r
	"68781757cc58563fb2ea9243822fcb3b973461be164b8d57f986146d16f2ee5e" + // s
	"1ccbe91c075fc7f4f033bfa248db8fccd3565de94bbfb12f3c59ff46c271bf83" + // pubX
	"ce4014c68811f9a21a1fdb2c0e6113e06db7ca93b7404e78dc7ccd5ca89a4ca9" //   pubY

// p256InvalidInput is 160 bytes with a real pubkey but zeroed r and s, so the
// signature cannot verify.
const p256InvalidInput = "0x" +
	"0000000000000000000000000000000000000000000000000000000000000001" + // hash
	"0000000000000000000000000000000000000000000000000000000000000000" + // r = 0
	"0000000000000000000000000000000000000000000000000000000000000000" + // s = 0
	"0000f56db78ca460b055c500064824bed999a25aaf48ebb519ef40f49be5a3fb" + // pubX
	"1549ee6e408c2e9e2b85b02b48a2f4f63f1a53ad77ce3de7b8d7a1d4aac39d82" //   pubY

func init() {
	testkit.Register(testkit.Case{
		Name:         "secp256r1-precompile-valid",
		Category:     "accounts",
		ChainCompat:  []string{"wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           secp256r1Valid,
	})
	testkit.Register(testkit.Case{
		Name:         "secp256r1-precompile-invalid",
		Category:     "accounts",
		ChainCompat:  []string{"wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           secp256r1Invalid,
	})
	testkit.Register(testkit.Case{
		Name:         "secp256r1-precompile-short-input",
		Category:     "accounts",
		ChainCompat:  []string{"wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           secp256r1ShortInput,
	})
}

// p256Call issues the precompile eth_call and returns the normalized result.
func p256Call(t *testkit.T, data string) string {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	c := rpc.Dial(primary.RPCURL)
	out, err := c.EthCall(t.Ctx(), p256VerifyAddr, data)
	t.NoErr(err, "eth_call secp256r1 precompile")
	return strings.ToLower(strings.TrimSpace(out))
}

// p256Empty reports whether the precompile returned no output (verification
// failure): geth returns "0x" (or an empty string) rather than a zero word.
func p256Empty(out string) bool {
	return out == "" || out == "0x"
}

func secp256r1Valid(t *testkit.T) {
	out := p256Call(t, p256ValidInput)
	t.Equalf(out, p256VerifyOne, "valid P-256 signature must verify (got %q)", out)
}

func secp256r1Invalid(t *testkit.T) {
	out := p256Call(t, p256InvalidInput)
	// Must NOT report success. geth returns empty on failure; some clients
	// return a zero word — both are acceptable, only 0x..01 is a bug.
	t.Truef(out != p256VerifyOne, "invalid signature must not verify (got %q)", out)
}

func secp256r1ShortInput(t *testkit.T) {
	// 159 bytes: one byte short of the required 160.
	short := "0x" + strings.Repeat("0", 318)
	out := p256Call(t, short)
	t.Truef(p256Empty(out), "159-byte input must return empty output (got %q)", out)

	// Empty input is likewise rejected.
	out = p256Call(t, "0x")
	t.Truef(p256Empty(out), "empty input must return empty output (got %q)", out)
}
