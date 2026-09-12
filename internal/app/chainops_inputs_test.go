package app

import (
	"context"
	"math/big"
	"strings"
	"testing"
)

// The value and bytecode decoders are shared by every write this layer makes —
// a transfer, a faucet top-up, a contract creation. They were uncovered, and
// two inputs got through that cannot have been meant.
//
// Both are the same shape: not a crash, and not a refusal. A negative amount
// travelled to the node, which then complained about encoding rather than about
// the sum the operator typed; "0x" bytecode produced a successful deploy of
// nothing, reporting an address for a contract that holds no code.

func TestWei_DecisionTable(t *testing.T) {
	cases := []struct {
		in    string
		want  int64
		errIs string
	}{
		{in: "", want: 0},
		{in: "0", want: 0},
		{in: " 7 ", want: 7},
		{in: "1000000000000000000", want: 1000000000000000000},
		// Decimal, not hex: the same digits in another base move a different sum.
		{in: "0x10", errIs: "decimal wei expected"},
		{in: "1_000", errIs: "decimal wei expected"},
		{in: "abc", errIs: "decimal wei expected"},
		// A negative amount is not a small one.
		{in: "-5", errIs: "cannot be negative"},
		{in: "-1000000000000000000", errIs: "cannot be negative"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := Wei(c.in)
			if c.errIs != "" {
				if err == nil {
					t.Fatalf("Wei(%q) = %v, wanted a refusal", c.in, got)
				}
				if !strings.Contains(err.Error(), c.errIs) {
					t.Errorf("refusal %q does not say %q", err, c.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wei(%q): %v", c.in, err)
			}
			if got.Cmp(big.NewInt(c.want)) != 0 {
				t.Errorf("Wei(%q) = %v, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestHexBytes_DecisionTable(t *testing.T) {
	cases := []struct {
		in    string
		want  int // decoded length
		errIs string
	}{
		{in: "", want: 0},
		// "0x" decodes to nothing and is not an error: an empty calldata is a
		// legitimate thing to send. A CALLER that needs bytes has to say so.
		{in: "0x", want: 0},
		{in: "0xabcd", want: 2},
		{in: "abcd", want: 2},
		{in: " 0xabcd ", want: 2},
		{in: "0xzz", errIs: "invalid byte"},
		{in: "0xabc", errIs: "odd length"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := HexBytes(c.in)
			if c.errIs != "" {
				if err == nil {
					t.Fatalf("HexBytes(%q) = %x, wanted a refusal", c.in, got)
				}
				if !strings.Contains(err.Error(), c.errIs) {
					t.Errorf("refusal %q does not say %q", err, c.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("HexBytes(%q): %v", c.in, err)
			}
			if len(got) != c.want {
				t.Errorf("HexBytes(%q) decoded %d bytes, want %d", c.in, len(got), c.want)
			}
		})
	}
}

// TestContractDeploy_RefusesBytecodeThatIsNoBytecode is the caller that needs
// bytes saying so. "0x" passes an emptiness check on the string and decodes to
// nothing, so the creation used to succeed and report an address for a contract
// with no code in it.
func TestContractDeploy_RefusesBytecodeThatIsNoBytecode(t *testing.T) {
	for _, code := range []string{"", "0x", " 0x "} {
		out, err := ContractDeploy(context.Background(), Deps{}, ContractDeployIn{
			Bytecode: code, FromKey: "0x" + strings.Repeat("11", 32),
			Chain: ChainRef{Chain: "wbft", RPC: "http://127.0.0.1:1"},
		})
		if err == nil {
			t.Errorf("bytecode %q deployed and reported %+v", code, out)
			continue
		}
		if !strings.Contains(err.Error(), "bytecode") {
			t.Errorf("the refusal for %q should name the bytecode: %v", code, err)
		}
	}
}

// TestContractDeploy_RefusesABadDeployerKeyBeforeDialling: the refusals are
// ordered so that a mistake in the arguments is reported as such, rather than as
// whatever the node says when reached with nonsense.
func TestContractDeploy_RefusesABadDeployerKeyBeforeDialling(t *testing.T) {
	_, err := ContractDeploy(context.Background(), Deps{}, ContractDeployIn{
		Bytecode: "0x6000", FromKey: "not-hex",
		Chain: ChainRef{Chain: "wbft", RPC: "http://127.0.0.1:1"},
	})
	if err == nil || !strings.Contains(err.Error(), "bad deployer key") {
		t.Errorf("a malformed key should be named as such: %v", err)
	}
}
